package manager

import (
	"time"

	"github.com/golang/protobuf/proto"
)

type Game interface {
	InitState() proto.Message
	DecodeState(data []byte) (proto.Message, error)
	DecodeAction(data []byte) (proto.Message, error)
	CheckAction(tickInfo *TickInfo, action proto.Message) error
	ApplyActions(tickInfo *TickInfo, actions []Action) *TickResult
	SmartGuyTurn(tickInfo *TickInfo) proto.Message
}

type TickInfo struct {
	Id        uint16
	DebugGame bool
	Finished  time.Time
	CurUid    uint32
	Uids      []uint32
	State     proto.Message
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
