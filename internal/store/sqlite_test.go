package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/MamadoubarryGLRSB/urlwatch/internal/domain"
	"github.com/MamadoubarryGLRSB/urlwatch/internal/store"
)

func TestSQLiteSaveAndGet(t *testing.T) {
	path := t.TempDir() + "/test.db"
	s, err := store.NewSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	batch := domain.Batch{
		ID:        "b_test01",
		CreatedAt: time.Now().UTC(),
		Status:    domain.StatusDone,
		Results: []domain.CheckResult{
			{URL: "https://go.dev", OK: true, StatusCode: 200, LatencyMS: 50},
		},
		Summary: domain.Summary{Total: 1, Up: 1, Down: 0, DurationMS: 50},
	}

	if err := s.Save(context.Background(), batch); err != nil {
		t.Fatal(err)
	}

	got, err := s.Get(context.Background(), "b_test01")
	if err != nil {
		t.Fatal(err)
	}
	if got.Summary.Up != 1 {
		t.Errorf("up attendu 1, reçu %d", got.Summary.Up)
	}
}

func TestSQLiteListFilter(t *testing.T) {
	path := t.TempDir() + "/test.db"
	s, err := store.NewSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	defer os.Remove(path)

	_ = s.Save(context.Background(), domain.Batch{ID: "b1", CreatedAt: time.Now(), Status: domain.StatusPending})
	_ = s.Save(context.Background(), domain.Batch{ID: "b2", CreatedAt: time.Now(), Status: domain.StatusDone, Summary: domain.Summary{Total: 1}})

	res, err := s.List(context.Background(), domain.ListParams{Status: domain.StatusDone, Page: 1, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 {
		t.Errorf("total attendu 1, reçu %d", res.Total)
	}
}
