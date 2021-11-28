package manager

import (
	"math/rand"
	"sync"
)

type Queue struct {
	mtx          sync.Mutex
	participants map[uint32]*QueueParticipant
}

type QueueParticipant struct {
	user *User
	ch   chan *GameInfo
}

func NewQueue() *Queue {
	return &Queue{
		participants: map[uint32]*QueueParticipant{},
	}
}

func (q *Queue) Join(user *User) (<-chan *GameInfo, error) {
	q.mtx.Lock()
	defer q.mtx.Unlock()

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

		delete(q.participants, participants[i*2].user.Id)
		delete(q.participants, participants[i*2+1].user.Id)
	}

	return res
}
