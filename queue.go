package manager

import (
	"github.com/prometheus/client_golang/prometheus"
	"math/rand"
	"sync"
)

type Queue struct {
	mtx            sync.Mutex
	participants   map[uint32]*QueueParticipant
	gaugeQueueSize prometheus.Gauge
}

type QueueParticipant struct {
	user *User
	ch   chan *GameInfo
}

func NewQueue(id string) *Queue {
	return &Queue{
		participants: map[uint32]*QueueParticipant{},
		gaugeQueueSize: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "game_manager.queue",
			Subsystem: id,
			Name:      "size",
		}),
	}
}

func (q *Queue) Join(user *User) (<-chan *GameInfo, error) {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	q.gaugeQueueSize.Inc()

	if participant, exists := q.participants[user.Id]; exists {
		close(participant.ch)
	}

	participant := &QueueParticipant{
		user: user,
		ch:   make(chan *GameInfo),
	}
	q.participants[user.Id] = participant

	return participant.ch, nil
}

func (q *Queue) Leave(user *User) {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	q.gaugeQueueSize.Dec()
	delete(q.participants, user.Id)
}

func (q *Queue) GetPairs() [][2]*QueueParticipant {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	participants := make([]*QueueParticipant, 0, len(q.participants))
	for _, participant := range q.participants {
		participants = append(participants, participant)
	}

	rand.Shuffle(len(participants), func(i, j int) {
		participants[i], participants[j] = participants[j], participants[i]
	})

	var res [][2]*QueueParticipant

	for i := 0; i < len(participants)/2; i++ {
		res = append(res, [2]*QueueParticipant{participants[i*2], participants[i*2+1]})

		q.gaugeQueueSize.Dec()
		delete(q.participants, participants[i*2].user.Id)

		q.gaugeQueueSize.Dec()
		delete(q.participants, participants[i*2+1].user.Id)
	}

	return res
}
