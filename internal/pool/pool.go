package pool

import (
	"context"
	"sync"
	"time"

	"github.com/MamadoubarryGLRSB/urlwatch/internal/domain"
)

// Run lance les checks en parallèle, borné par opts.Concurrency.
func Run(ctx context.Context, checker domain.Checker, urls []string, opts domain.CheckOptions) []domain.CheckResult {
	if len(urls) == 0 {
		return nil
	}

	// channels bufferisés pour éviter les blocages entre workers
	jobs := make(chan string, len(urls))
	results := make(chan domain.CheckResult, len(urls))

	urlTimeout := time.Duration(opts.TimeoutMS) * time.Millisecond

	var wg sync.WaitGroup

	for i := 0; i < opts.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for url := range jobs {
				// si le lot est annulé on arrête sans appeler le checker
				select {
				case <-ctx.Done():
					results <- domain.CheckResult{
						URL:   url,
						OK:    false,
						Error: ctx.Err().Error(),
					}
					continue
				default:
				}

				urlCtx, cancel := context.WithTimeout(ctx, urlTimeout)
				r := checker.Check(urlCtx, url)
				cancel()

				results <- r
			}
		}()
	}

	for _, u := range urls {
		jobs <- u
	}
	close(jobs)

	// on ferme results une fois tous les workers finis
	go func() {
		wg.Wait()
		close(results)
	}()

	allResults := make([]domain.CheckResult, 0, len(urls))
	for r := range results {
		allResults = append(allResults, r)
	}

	return allResults
}
