package manager

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
)

type GameManager struct {
	game      Game
	storage   Storage
	queue     *Queue
	scheduler Scheduler
	games     map[uuid.UUID]*gameInstance
	gamesMtx  sync.RWMutex
}

type GameInfo struct {
	Uuid  uuid.UUID
	Debug bool
	Uids  []uint32
}

func New(game Game, storage Storage, scheduler Scheduler) *GameManager {
	m := &GameManager{
		game:      game,
		storage:   storage,
		queue:     NewQueue(),
		scheduler: scheduler,
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
		Uuid:  g.uuid,
		Debug: debug,
		Uids:  g.uids,
	}

	err := m.storage.CreateGame(ctx, res)
	if err != nil {
		return nil, err
	}

	g.Start()

	return res, nil
}

func (m *GameManager) JoinGame(ctx context.Context, token string, debug bool) (*GameInfo, error) {
	user, err := m.storage.GetUserByToken(ctx, token)
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
