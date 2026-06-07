package app

import "sync"

type EventHub struct {
	mu   sync.Mutex
	subs map[string]map[chan *Task]struct{}
}

func NewEventHub() *EventHub {
	return &EventHub{subs: make(map[string]map[chan *Task]struct{})}
}

func (h *EventHub) Subscribe(taskID string) (chan *Task, func()) {
	ch := make(chan *Task, 8)
	h.mu.Lock()
	if h.subs[taskID] == nil {
		h.subs[taskID] = make(map[chan *Task]struct{})
	}
	h.subs[taskID][ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		if h.subs[taskID] != nil {
			delete(h.subs[taskID], ch)
			if len(h.subs[taskID]) == 0 {
				delete(h.subs, taskID)
			}
		}
		close(ch)
		h.mu.Unlock()
	}
}

func (h *EventHub) Publish(taskID string, task *Task) {
	if h == nil || task == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[taskID] {
		select {
		case ch <- cloneTask(task):
		default:
		}
	}
}
