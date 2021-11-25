package manager

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
)

type GameInfo struct {
	Id   uint32
	Uuid uuid.UUID
	Uids []uint32
}

type GameManager interface {
	CreateGame(ctx context.Context, uids []uint32, debug bool) (*GameInfo, error)
	JoinGame(ctx context.Context, token string, debug bool) (*GameInfo, error)
	WaitTurn(ctx context.Context, userToken, gameUuid string) (*TickInfo, error)
	DoAction(ctx context.Context, userToken, gameUuid string, action proto.Message) error
}
