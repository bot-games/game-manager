package manager

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

const turnTimeout = 5 * time.Second

type gameInstance struct {
	game          Game
	uuid          uuid.UUID
	uids          []uint32
	mtx           *sync.Mutex
	options       proto.Message
	data          any
	ticks         []Tick
	waitUsers     uint8
	timeoutTimer  *time.Timer
	waitUsersCond *sync.Cond
	debug         bool
	finished      time.Time
	winner        uint8
	timeout       uint8
	onFinish      func(g *gameInstance)
}

type Tick struct {
	State   proto.Message
	Actions []Action
}

func NewG(game Game, uids []uint32, debug bool, onFinish func(g *gameInstance)) *gameInstance {
	mtx := &sync.Mutex{}

	options, state, waitUsers, gameData := game.Init()

	return &gameInstance{
		game:          game,
		uuid:          uuid.New(),
		uids:          uids,
		mtx:           mtx,
		options:       options,
		data:          gameData,
		ticks:         []Tick{{State: state}},
		waitUsers:     waitUsers,
		waitUsersCond: sync.NewCond(mtx),
		debug:         debug,
		onFinish:      onFinish,
	}
}

func (g *gameInstance) GetUuid() uuid.UUID {
	return g.uuid
}

func (g *gameInstance) Start() {
	if !g.debug {
		g.timeoutTimer = time.AfterFunc(turnTimeout, func() {
			g.mtx.Lock()
			defer g.mtx.Unlock()
			defer g.waitUsersCond.Broadcast()

			if g.waitUsers == 0 {
				return
			}

			g.finished = time.Now()
			g.timeout = g.waitUsers
			g.winner = 0
			switch g.waitUsers {
			case 1:
				g.winner = 2
			case 2:
				g.winner = 1
			}

			g.onFinish(g)
		})
	}

	if g.debug {
		go func() {
			for {
				tickInfo, err := g.WaitTurn(context.Background(), 0)
				if err != nil {
					break
				}

				if err := g.DoAction(0, g.game.SmartGuyTurn(tickInfo)); err != nil {
					log.Printf("Cannot process SmartGuy action: %v", err)
				}
			}
		}()
	}
}

func (g *gameInstance) WaitTurn(ctx context.Context, uid uint32) (*TickInfo, error) {
	userMask := g.getUserMask(uid)

	g.mtx.Lock()
	defer g.mtx.Unlock()

	for g.finished.IsZero() && g.waitUsers&userMask == 0 {
		g.waitUsersCond.Wait()
	}

	if !g.finished.IsZero() {
		return nil, ErrEndOfGame{Winner: g.winner, IsYou: append([]uint32{0}, g.uids...)[g.winner] == uid}
	}

	return g.getCurTick(uid), nil
}

func (g *gameInstance) DoAction(uid uint32, action proto.Message) error {
	userMask := g.getUserMask(uid)

	g.mtx.Lock()
	defer g.mtx.Unlock()

	if !g.finished.IsZero() {
		return ErrGameFinished
	}

	if g.waitUsers&userMask == 0 {
		return ErrInvalidAction
	}

	if err := g.game.CheckAction(g.getCurTick(uid), action); err != nil {
		return err
	}

	g.waitUsers &= ^userMask

	g.ticks[len(g.ticks)-1].Actions = append(g.ticks[len(g.ticks)-1].Actions, Action{
		Uid:    uid,
		Action: action,
	})

	if g.waitUsers == 0 {
		go g.processCurTick()
	}

	return nil
}

func (g *gameInstance) getCurTick(uid uint32) *TickInfo {
	return &TickInfo{
		Id:          uint16(len(g.ticks) - 1),
		DebugGame:   g.debug,
		GameOptions: g.options,
		Finished:    g.finished,
		CurUid:      uid,
		Uids:        g.uids,
		State:       proto.Clone(g.ticks[len(g.ticks)-1].State),
		GameData:    g.data,
	}
}

func (g *gameInstance) processCurTick() {
	g.mtx.Lock()
	defer g.mtx.Unlock()
	defer g.waitUsersCond.Broadcast()

	if g.timeoutTimer != nil {
		g.timeoutTimer.Stop()
	}

	nextTick := g.game.ApplyActions(g.getCurTick(0), g.ticks[len(g.ticks)-1].Actions)
	g.ticks = append(g.ticks, Tick{
		State: nextTick.NewState,
	})

	if nextTick.GameFinished {
		g.finished = time.Now()
		g.winner = nextTick.Winner

		g.onFinish(g)

		return
	}

	g.waitUsers = nextTick.NextTurnPlayers

	if !g.debug {
		g.timeoutTimer.Reset(turnTimeout)
	}
}

func (g *gameInstance) getUserMask(uid uint32) uint8 {
	mask := uint8(1)
	for i, u := range g.uids {
		if u == uid {
			if i > 0 {
				mask *= 2
			}

			break
		}
	}

	return mask
}
