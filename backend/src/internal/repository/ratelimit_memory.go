package repository

import (
	"sync"
	"time"
)

type InMemoryRateLimitRepository struct {
	mu      sync.Mutex
	entries map[string][]time.Time
}

func NewInMemoryRateLimitRepository() *InMemoryRateLimitRepository {
	return &InMemoryRateLimitRepository{entries: make(map[string][]time.Time)}
}

func (r *InMemoryRateLimitRepository) mapKey(limitType, key string) string {
	return limitType + ":" + key
}

func (r *InMemoryRateLimitRepository) GetRequests(limitType, key string, since time.Time) ([]time.Time, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := r.mapKey(limitType, key)
	all := r.entries[k]
	var result []time.Time
	for _, t := range all {
		if t.After(since) {
			result = append(result, t)
		}
	}
	return result, nil
}

func (r *InMemoryRateLimitRepository) Record(limitType, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := r.mapKey(limitType, key)
	r.entries[k] = append(r.entries[k], time.Now())
	return nil
}
