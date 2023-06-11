package manager

import (
	"time"

	"google.golang.org/protobuf/proto"
)

type Game interface {
	Init() (options proto.Message, state proto.Message, waitUsers uint8, gameData any)
	//DecodeState(data []byte) (proto.Message, error)
	//DecodeAction(data []byte) (proto.Message, error)
	CheckAction(tickInfo *TickInfo, action proto.Message) error
	ApplyActions(tickInfo *TickInfo, actions []Action) *TickResult
	SmartGuyTurn(tickInfo *TickInfo) proto.Message
}

type TickInfo struct {
	Id          uint16
	DebugGame   bool
	GameOptions proto.Message
	Finished    time.Time
	CurUid      uint32
	Uids        []uint32
	State       proto.Message
	GameData    any
}

type Action struct {
	Uid    uint32
	Action proto.Message
}

type TickResult struct {
	GameFinished    bool
	Winner          uint8
	NewState        proto.Message
	NextTurnPlayers uint8
}
