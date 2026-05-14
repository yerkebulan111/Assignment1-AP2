package repository

import "sync"

type IdempotencyStore interface {
	IsProcessed(eventID string) bool
	MarkProcessed(eventID string)
}

type InMemoryIdempotencyStore struct {
	mu   sync.RWMutex
	seen map[string]struct{}
}

func NewInMemoryIdempotencyStore() IdempotencyStore {
	return &InMemoryIdempotencyStore{seen: make(map[string]struct{})}
}

func (s *InMemoryIdempotencyStore) IsProcessed(eventID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.seen[eventID]
	return ok
}

func (s *InMemoryIdempotencyStore) MarkProcessed(eventID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen[eventID] = struct{}{}
}
