package dedup

import (
	"sync"
	"time"
)

type IDeduplication interface {
	Deduplicate(key string) bool
}

type Deduplication struct {
	keys    map[string]time.Time
	keysTTL time.Duration
	mu      sync.Mutex
}

func NewDeduplicationInstance() *Deduplication {
	instance := &Deduplication{
		keys:    make(map[string]time.Time),
		keysTTL: time.Minute,
		mu:      sync.Mutex{},
	}

	instance.runWorker()

	return instance
}

// Deduplicate removes duplicate keys from the Deduplication map and returns true if the key was already present.
//
// key string
// bool
func (d *Deduplication) Deduplicate(key string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.keys[key]; ok {
		return true
	}
	d.keys[key] = time.Now()
	return false
}

func (d *Deduplication) runWorker() {
	ticker := time.Tick(time.Second * 10)

	go func() {
		for {
			<-ticker
			d.mu.Lock()
			for key, value := range d.keys {
				if time.Since(value) > d.keysTTL {
					delete(d.keys, key)
				}
			}
			d.mu.Unlock()
		}
	}()
}
