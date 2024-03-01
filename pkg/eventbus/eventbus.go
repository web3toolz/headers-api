package eventbus

import (
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"sync"
)

type IEventBus interface {
	Subscribe(topic string, id string, fn func(header *types.Header)) error
	Unsubscribe(topic string, id string)
	Publish(topic string, args *types.Header)
}

type EventBus struct {
	m  map[string]map[string]*func(header *types.Header)
	mu sync.Mutex
	wg sync.WaitGroup
}

func New() *EventBus {
	return &EventBus{
		m:  make(map[string]map[string]*func(header *types.Header)),
		mu: sync.Mutex{},
		wg: sync.WaitGroup{},
	}
}

func (b *EventBus) Publish(topic string, header *types.Header) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if topicMap, ok := b.m[topic]; ok {
		for _, callback := range topicMap {
			callback_ := *callback
			b.wg.Add(1)
			go func() {
				callback_(header)
				b.wg.Done()
			}()
		}
	}
}

func (b *EventBus) Subscribe(topic string, id string, fn func(header *types.Header)) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if topicMap, ok := b.m[topic]; ok {
		if _, ok2 := topicMap[id]; ok2 {
			return fmt.Errorf("callback already registered")
		} else {
			topicMap[id] = &fn
		}
	} else {
		b.m[topic] = make(map[string]*func(header *types.Header))
		b.m[topic][id] = &fn
	}
	return nil
}

func (b *EventBus) Unsubscribe(topic string, id string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if topicMap, ok := b.m[topic]; ok {
		if _, ok2 := topicMap[id]; ok2 {
			delete(topicMap, id)

			if len(topicMap) == 0 {
				delete(b.m, topic)
			}
		}
	}
	return
}

func (b *EventBus) Wait() {
	b.wg.Wait()
}
