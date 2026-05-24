package sse

import (
	"context"
	"log"
	"sync"
)

type Event struct {
	Step        string `json:"step"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	Percent     int    `json:"percent,omitempty"`
	Details     string `json:"details,omitempty"`
	Volume      string `json:"volume,omitempty"`
	VolumeIndex int    `json:"volume_index,omitempty"`
	VolumeTotal int    `json:"volume_total,omitempty"`
}

type Broadcaster struct {
	mu          sync.RWMutex
	subscribers map[string][]chan Event
	done        map[string]chan struct{}
	buffer      map[string][]Event
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		subscribers: make(map[string][]chan Event),
		done:        make(map[string]chan struct{}),
		buffer:      make(map[string][]Event),
	}
}

// Register pre-registers a token so events are buffered until a subscriber connects.
func (b *Broadcaster) Register(token string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.buffer[token]; !ok {
		b.buffer[token] = nil
	}
	if _, ok := b.done[token]; !ok {
		b.done[token] = make(chan struct{})
	}
	log.Printf("[sse] registered token %s", token)
}

func (b *Broadcaster) Subscribe(token string) chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan Event, 64)

	// Replay buffered events to the new subscriber.
	if buffered, ok := b.buffer[token]; ok {
		for _, event := range buffered {
			select {
			case ch <- event:
			default:
				log.Printf("[sse] warning: dropped buffered event for token %s (channel full)", token)
			}
		}
		delete(b.buffer, token)
	}

	b.subscribers[token] = append(b.subscribers[token], ch)

	if _, ok := b.done[token]; !ok {
		b.done[token] = make(chan struct{})
	}

	log.Printf("[sse] subscriber connected for token %s", token)
	return ch
}

func (b *Broadcaster) Unsubscribe(token string, ch chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subs := b.subscribers[token]
	for i, s := range subs {
		if s == ch {
			b.subscribers[token] = append(subs[:i], subs[i+1:]...)
			close(ch)
			return
		}
	}
}

func (b *Broadcaster) Send(token string, event Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subs := b.subscribers[token]
	if len(subs) == 0 {
		// No subscribers yet — buffer the event if the token was registered.
		if _, registered := b.buffer[token]; registered {
			b.buffer[token] = append(b.buffer[token], event)
			log.Printf("[sse] buffered event for token %s: step=%s", token, event.Step)
			return
		}
		// Token not registered either — event is lost (shouldn't happen with Register).
		if _, exists := b.done[token]; exists {
			b.buffer[token] = append(b.buffer[token], event)
			return
		}
		return
	}

	for _, ch := range subs {
		select {
		case ch <- event:
		default:
		}
	}
}

func (b *Broadcaster) Complete(token string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	log.Printf("[sse] completing token %s", token)
	if ch, ok := b.done[token]; ok {
		close(ch)
	}
}

func (b *Broadcaster) Wait(ctx context.Context, token string) {
	b.mu.RLock()
	ch, ok := b.done[token]
	b.mu.RUnlock()

	if !ok {
		return
	}

	select {
	case <-ch:
	case <-ctx.Done():
	}
}

func (b *Broadcaster) Cleanup(token string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, ch := range b.subscribers[token] {
		close(ch)
	}
	delete(b.subscribers, token)
	delete(b.done, token)
	delete(b.buffer, token)
}
