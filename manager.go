package manager

import (
	"context"

	"github.com/google/uuid"
)

type GameInfo struct {
	Id   uint32
	Uuid uuid.UUID
	Uid1 uint32
	Uid2 uint32
}

type GameManager interface {
	CreateGame(ctx context.Context, uid1, uid2 uint32, debug bool) (*GameInfo, error)
	JoinGame(ctx context.Context, token string, debug bool) (*GameInfo, error)
	WaitTurn(ctx context.Context, userToken, gameUuid string) (*TickInfo, error)
	DoAction(ctx context.Context, userToken, gameUuid string, action string, data interface{}) error
}
