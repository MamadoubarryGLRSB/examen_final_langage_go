package store

import (
	"context"
	"fmt"
	"sync"

	"github.com/MamadoubarryGLRSB/urlwatch/internal/domain"
)

// stockage en mémoire, perdu au redémarrage
type MemoryStore struct {
	mu      sync.RWMutex
	batches map[string]domain.Batch
}

func NewMemory() *MemoryStore {
	return &MemoryStore{
		batches: make(map[string]domain.Batch),
	}
}

func (s *MemoryStore) Save(_ context.Context, b domain.Batch) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.batches[b.ID] = b
	return nil
}

func (s *MemoryStore) Get(_ context.Context, id string) (domain.Batch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.batches[id]
	if !ok {
		return domain.Batch{}, fmt.Errorf("store.Get %q: %w", id, domain.ErrBatchNotFound)
	}
	return b, nil
}
