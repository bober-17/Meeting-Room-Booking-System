package hub

import (
	"fmt"
	"sync"

	"github.com/bober-17/meeting-room-booking-system/notification-service/internal/model"
)

const maxSubsPerUser = 10

type Hub struct {
	mu   sync.Mutex
	subs map[string][]chan model.Notification
}

func NewHub() *Hub {
	return &Hub{
		subs: make(map[string][]chan model.Notification),
	}
}

func (h *Hub) Subscribe(userID string) (<-chan model.Notification, func(), error) {
	h.mu.Lock()
	if len(h.subs[userID]) >= maxSubsPerUser {
		h.mu.Unlock()
		return nil, nil, fmt.Errorf("too many active SSE connections for user")
	}
	ch := make(chan model.Notification, 16)
	h.subs[userID] = append(h.subs[userID], ch)
	h.mu.Unlock()

	unsubscribe := func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		chans := h.subs[userID]
		for i, c := range chans {
			if c == ch {
				last := len(chans) - 1
				chans[i] = chans[last]
				chans[last] = nil
				h.subs[userID] = chans[:last]
				break
			}
		}
		if len(h.subs[userID]) == 0 {
			delete(h.subs, userID)
		}
	}

	return ch, unsubscribe, nil
}

func (h *Hub) Broadcast(userID string, n model.Notification) {
	h.mu.Lock()
	chans := make([]chan model.Notification, len(h.subs[userID]))
	copy(chans, h.subs[userID])
	h.mu.Unlock()

	for _, ch := range chans {
		select {
		case ch <- n:
		default:
		}
	}
}
