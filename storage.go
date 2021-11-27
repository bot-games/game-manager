package manager

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Storage interface {
	GetUserByToken(ctx context.Context, token string) (*User, error)
	CreateGame(ctx context.Context, info *GameInfo) error
	SaveGame(ctx context.Context, uuid uuid.UUID, winner, timeout uint8, finished time.Time, ticks []Tick) error
}

type User struct {
	Id    uint32
	Score uint32
}
