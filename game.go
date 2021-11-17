package manager

import (
	"time"

	"github.com/golang/protobuf/proto"
)

type Game interface {
	InitState() proto.Message
	GetActions(tickInfo *TickInfo) []string
	DoAction(tickInfo *TickInfo, action string, data interface{}) (proto.Message, error)
	IsGameFinished(tickInfo *TickInfo) (bool, uint8, error)
	SmartGuyTurn(tickInfo *TickInfo) (string, interface{})
}

type TickInfo struct {
	Id        uint16
	GameId    uint32
	DebugGame bool
	Finished  time.Time
	Winner    uint8
	WaitUsers uint8
	CurUid    uint32
	Uid1      uint32
	Uid2      uint32
	Data      []byte
}
