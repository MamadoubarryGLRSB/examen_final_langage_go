package checker

import (
	"context"
	"net/http"
	"time"

	"github.com/MamadoubarryGLRSB/urlwatch/internal/domain"
)

type HTTPChecker struct {
	client *http.Client
}

func New() *HTTPChecker {
	return &HTTPChecker{
		client: &http.Client{
			// on ne suit pas les redirections
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (c *HTTPChecker) Check(ctx context.Context, url string) domain.CheckResult {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return domain.CheckResult{
			URL:       url,
			OK:        false,
			LatencyMS: time.Since(start).Milliseconds(),
			Error:     err.Error(),
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return domain.CheckResult{
			URL:       url,
			OK:        false,
			LatencyMS: time.Since(start).Milliseconds(),
			Error:     cleanError(err),
		}
	}
	defer resp.Body.Close()

	latency := time.Since(start).Milliseconds()
	ok := resp.StatusCode >= 200 && resp.StatusCode < 400

	return domain.CheckResult{
		URL:        url,
		StatusCode: resp.StatusCode,
		OK:         ok,
		LatencyMS:  latency,
	}
}

func cleanError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
