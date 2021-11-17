package manager

import (
	"errors"
	"fmt"
)

type ErrInGame string

func (e ErrInGame) Error() string {
	return fmt.Sprintf("user is alredy in game: %s", e)
}

type ErrEndOfGame struct {
	Winner uint8
	IsYou  bool
}

func (e ErrEndOfGame) Error() string {
	if e.Winner == 0 {
		return fmt.Sprintf("the game was finished, the result is draw")
	}

	return fmt.Sprintf("the game was finished, the winner is participant #%d", e.Winner)
}

var (
	ErrInvalidToken  = errors.New("invalid token")
	ErrInQueue       = errors.New("the user is already in queue")
	ErrInvalidGameId = errors.New("invalid game ID")
	ErrInvalidAction = errors.New("invalid action")
)
