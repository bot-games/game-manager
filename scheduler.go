package manager

type Scheduler interface {
	Notify(ev SchedulerEvent)
	SetOnReady(callback func())
}

type SchedulerEvent uint8

const (
	SchedulerEventJoin SchedulerEvent = iota
	SchedulerEventLeave
)
