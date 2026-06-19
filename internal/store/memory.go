package store

import (
	"context"
	"fmt"
	"sort"
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

func (s *MemoryStore) List(_ context.Context, p domain.ListParams) (domain.ListResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]domain.Batch, 0, len(s.batches))
	for _, b := range s.batches {
		if p.Status != "" && b.Status != p.Status {
			continue
		}
		all = append(all, b)
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})

	total := len(all)
	start := (p.Page - 1) * p.Limit
	if start >= total {
		return domain.ListResult{Items: []domain.Batch{}, Page: p.Page, Limit: p.Limit, Total: total}, nil
	}
	end := start + p.Limit
	if end > total {
		end = total
	}

	return domain.ListResult{
		Items: all[start:end],
		Page:  p.Page,
		Limit: p.Limit,
		Total: total,
	}, nil
}
