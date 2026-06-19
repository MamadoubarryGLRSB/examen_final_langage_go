package pool_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/MamadoubarryGLRSB/urlwatch/internal/checker"
	"github.com/MamadoubarryGLRSB/urlwatch/internal/domain"
	"github.com/MamadoubarryGLRSB/urlwatch/internal/pool"
)

func TestPoolBasic(t *testing.T) {
	mock := checker.NewMock(map[string]domain.CheckResult{
		"https://a.com": {OK: true, StatusCode: 200},
		"https://b.com": {OK: false, StatusCode: 500},
	})

	urls := []string{"https://a.com", "https://b.com"}
	opts := domain.CheckOptions{Concurrency: 2, TimeoutMS: 1000}

	results := pool.Run(context.Background(), mock, urls, opts)

	if len(results) != 2 {
		t.Fatalf("attendu 2 résultats, reçu %d", len(results))
	}
}

func TestPoolBoundedConcurrency(t *testing.T) {
	responses := map[string]domain.CheckResult{}
	for i := 0; i < 10; i++ {
		url := "https://url" + string(rune('a'+i)) + ".com"
		responses[url] = domain.CheckResult{OK: true, StatusCode: 200}
	}

	counting := &countingMock{
		inner: checker.NewMock(responses),
	}

	urls := make([]string, 10)
	for i := range urls {
		urls[i] = "https://url" + string(rune('a'+i)) + ".com"
	}

	opts := domain.CheckOptions{Concurrency: 3, TimeoutMS: 2000}
	results := pool.Run(context.Background(), counting, urls, opts)

	if len(results) != 10 {
		t.Fatalf("attendu 10 résultats, reçu %d", len(results))
	}
	if counting.maxActive > 3 {
		t.Errorf("concurrence bornée à 3, mais %d appels simultanés détectés", counting.maxActive)
	}
}

func TestPoolCancellation(t *testing.T) {
	slow := &slowMock{delay: 200 * time.Millisecond}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	urls := []string{"https://a.com", "https://b.com", "https://c.com"}
	opts := domain.CheckOptions{Concurrency: 2, TimeoutMS: 1000}

	start := time.Now()
	results := pool.Run(ctx, slow, urls, opts)
	elapsed := time.Since(start)

	if len(results) == 0 {
		t.Fatal("attendu au moins un résultat même en cas d'annulation")
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("le pool aurait dû s'arrêter rapidement, durée réelle : %v", elapsed)
	}
}

func TestPoolEmpty(t *testing.T) {
	mock := checker.NewMock(nil)
	opts := domain.CheckOptions{Concurrency: 4, TimeoutMS: 1000}
	results := pool.Run(context.Background(), mock, nil, opts)
	if results != nil {
		t.Errorf("attendu nil pour une liste vide, reçu %v", results)
	}
}

type countingMock struct {
	inner     domain.Checker
	mu        sync.Mutex // pour le test -race
	current   int
	maxActive int
}

func (c *countingMock) Check(ctx context.Context, url string) domain.CheckResult {
	c.mu.Lock()
	c.current++
	if c.current > c.maxActive {
		c.maxActive = c.current
	}
	c.mu.Unlock()

	time.Sleep(10 * time.Millisecond)

	c.mu.Lock()
	c.current--
	c.mu.Unlock()

	return c.inner.Check(ctx, url)
}

type slowMock struct {
	delay time.Duration
}

func (s *slowMock) Check(ctx context.Context, url string) domain.CheckResult {
	select {
	case <-time.After(s.delay):
		return domain.CheckResult{URL: url, OK: true, StatusCode: 200}
	case <-ctx.Done():
		return domain.CheckResult{URL: url, OK: false, Error: ctx.Err().Error()}
	}
}
