package manager

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-qbit/rpc/openapi"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

type GameManager struct {
	id        string
	caption   string
	game      Game
	storage   Storage
	queue     *Queue
	scheduler Scheduler
	gameApi   RpcCreator
	games     map[uuid.UUID]*gameInstance
	gamesMtx  sync.RWMutex
}

type GameInfo struct {
	Options proto.Message
	Uuid    uuid.UUID
	Debug   bool
	Uids    []uint32
}

type RpcCreator func(m *GameManager) GameApi

type GameApi interface {
	http.Handler
	GetSwagger(ctx context.Context) *openapi.OpenApi
	GetPlayerHandler() http.Handler
}

func New(id, caption string, game Game, storage Storage, scheduler Scheduler, rpc RpcCreator) *GameManager {
	m := &GameManager{
		id:        id,
		caption:   caption,
		game:      game,
		storage:   storage,
		queue:     NewQueue(),
		scheduler: scheduler,
		gameApi:   rpc,
		games:     map[uuid.UUID]*gameInstance{},
	}

	m.scheduler.SetOnReady(func() {
		for _, pair := range m.queue.GetPairs() {
			gameInfo, err := m.CreateGame(context.Background(), []uint32{pair[0].user.Id, pair[1].user.Id}, false)
			if err != nil {
				log.Printf("Cannot create game: %v", err)
			} else {
				for _, participant := range pair {
					participant.ch <- gameInfo
				}
			}
			for _, participant := range pair {
				close(participant.ch)
			}
		}
	})

	return m
}

func (m *GameManager) GetId() string       { return m.id }
func (m *GameManager) GetCaption() string  { return m.caption }
func (m *GameManager) GetGameApi() GameApi { return m.gameApi(m) }

func (m *GameManager) CreateGame(ctx context.Context, uids []uint32, debug bool) (*GameInfo, error) {
	g := NewG(m.game, uids, debug, func(g *gameInstance) {
		for i := 0; i < 10; i++ {
			if err := m.storage.SaveGame(ctx, g.uuid, g.winner, g.timeout, g.finished, g.ticks); err != nil {
				log.Printf("Cannot save game %s[%d]: %v", g.uuid.String(), i, err)
				time.Sleep(time.Duration(i+1) * 5 * time.Second)
				continue
			}
			break
		}
	})

	m.gamesMtx.Lock()
	defer m.gamesMtx.Unlock()

	// Clean old games
	now := time.Now()
	for id, g := range m.games {
		if !g.finished.IsZero() && now.Sub(g.finished) > time.Hour {
			delete(m.games, id)
		}
	}

	m.games[g.uuid] = g

	res := &GameInfo{
		Options: g.options,
		Uuid:    g.uuid,
		Debug:   debug,
		Uids:    g.uids,
	}

	err := m.storage.CreateGame(ctx, res)
	if err != nil {
		return nil, err
	}

	g.Start()

	return res, nil
}

func (m *GameManager) JoinGame(ctx context.Context, userToken string, debug bool) (*GameInfo, error) {
	user, err := m.storage.GetUserByToken(ctx, userToken)
	if err != nil {
		return nil, err
	}

	if debug {
		return m.CreateGame(ctx, []uint32{user.Id, 0}, true)
	}

	if g := m.getUserGame(user.Id); g != nil {
		return nil, ErrInGame(g.uuid.String())
	}

	ch, err := m.queue.Join(user)
	if err != nil {
		return nil, err
	}

	go m.scheduler.Notify(SchedulerEventJoin)

	select {
	case gi, ok := <-ch:
		if !ok {
			return nil, errors.New("something went wrong")
		}

		return gi, nil

	case <-ctx.Done():
		go m.scheduler.Notify(SchedulerEventLeave)
		m.queue.Leave(user)
		return nil, errors.New("disconnected")
	}
}

func (m *GameManager) RejoinGame(ctx context.Context, userToken, gameUuid string) (*GameInfo, error) {
	user, err := m.storage.GetUserByToken(ctx, userToken)
	if err != nil {
		return nil, err
	}

	g, err := m.getGame(gameUuid)
	if err != nil {
		return nil, err
	}

	if !g.finished.IsZero() {
		return nil, ErrEndOfGame{Winner: g.winner, IsYou: append([]uint32{0}, g.uids...)[g.winner] == user.Id}
	}

	return &GameInfo{
		Options: g.options,
		Uuid:    g.uuid,
		Debug:   g.debug,
		Uids:    g.uids,
	}, nil
}

func (m *GameManager) WaitTurn(ctx context.Context, userToken, gameUuid string) (*TickInfo, error) {
	user, err := m.storage.GetUserByToken(ctx, userToken)
	if err != nil {
		return nil, err
	}

	g, err := m.getGame(gameUuid)
	if err != nil {
		return nil, err
	}

	return g.WaitTurn(ctx, user.Id)
}

func (m *GameManager) DoAction(ctx context.Context, userToken, gameUuid string, action proto.Message) error {
	user, err := m.storage.GetUserByToken(ctx, userToken)
	if err != nil {
		return err
	}

	g, err := m.getGame(gameUuid)
	if err != nil {
		return err
	}

	return g.DoAction(user.Id, action)
}

func (m *GameManager) getGame(gameUuid string) (*gameInstance, error) {
	bGameUuid, err := uuid.Parse(gameUuid)
	if err != nil {
		return nil, ErrInvalidGameId
	}

	m.gamesMtx.RLock()
	defer m.gamesMtx.RUnlock()

	g, exists := m.games[bGameUuid]
	if !exists {
		return nil, ErrInvalidGameId
	}

	return g, nil
}

func (m *GameManager) getUserGame(uid uint32) *gameInstance {
	m.gamesMtx.RLock()
	defer m.gamesMtx.RUnlock()

	for _, g := range m.games {
		if !g.finished.IsZero() {
			continue
		}

		for _, gUid := range g.uids {
			if uid == gUid {
				return g
			}
		}
	}

	return nil
}
