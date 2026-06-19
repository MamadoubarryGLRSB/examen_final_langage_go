package domain

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type CheckResult struct {
	URL        string `json:"url"`
	StatusCode int    `json:"status_code,omitempty"`
	OK         bool   `json:"ok"`
	LatencyMS  int64  `json:"latency_ms"`
	Error      string `json:"error,omitempty"`
}

type Summary struct {
	Total      int   `json:"total"`
	Up         int   `json:"up"`
	Down       int   `json:"down"`
	DurationMS int64 `json:"duration_ms"`
}

type Batch struct {
	ID        string        `json:"batch_id"`
	CreatedAt time.Time     `json:"created_at"`
	Status    string        `json:"status,omitempty"`
	Results   []CheckResult `json:"results"`
	Summary   Summary       `json:"summary"`
}

type CheckOptions struct {
	Concurrency int `json:"concurrency"`
	TimeoutMS   int `json:"timeout_ms"`
}

type CheckRequest struct {
	URLs    []string     `json:"urls"`
	Options CheckOptions `json:"options"`
}

const (
	DefaultConcurrency = 8
	MaxConcurrency     = 50
	MinConcurrency     = 1

	DefaultTimeoutMS = 5000
	MinTimeoutMS     = 100
	MaxTimeoutMS     = 30000

	MaxURLs = 100

	StatusPending = "pending"
	StatusDone    = "done"

	DefaultListLimit = 20
	MaxListLimit     = 100
)

type Checker interface {
	Check(ctx context.Context, url string) CheckResult
}

type Store interface {
	Save(ctx context.Context, b Batch) error
	Get(ctx context.Context, id string) (Batch, error)
	List(ctx context.Context, p ListParams) (ListResult, error)
}

type ListParams struct {
	Status string
	Page   int
	Limit  int
}

type ListResult struct {
	Items []Batch `json:"items"`
	Page  int     `json:"page"`
	Limit int     `json:"limit"`
	Total int     `json:"total"`
}

var ErrBatchNotFound = errors.New("batch not found")

// erreur de validation avec le champ fautif
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("champ %q invalide : %s", e.Field, e.Message)
}

func ApplyDefaults(opts *CheckOptions) {
	if opts.Concurrency == 0 {
		opts.Concurrency = DefaultConcurrency
	}
	if opts.TimeoutMS == 0 {
		opts.TimeoutMS = DefaultTimeoutMS
	}
}

func ValidateRequest(req *CheckRequest) error {
	if len(req.URLs) == 0 {
		return &ValidationError{Field: "urls", Message: "au moins une URL est requise"}
	}
	if len(req.URLs) > MaxURLs {
		return &ValidationError{Field: "urls", Message: fmt.Sprintf("maximum %d URLs par lot", MaxURLs)}
	}
	for _, u := range req.URLs {
		if !isHTTPURL(u) {
			return &ValidationError{Field: "urls", Message: fmt.Sprintf("%q n'est pas une URL http/https valide", u)}
		}
	}
	if req.Options.Concurrency < MinConcurrency || req.Options.Concurrency > MaxConcurrency {
		return &ValidationError{
			Field:   "options.concurrency",
			Message: fmt.Sprintf("doit être entre %d et %d", MinConcurrency, MaxConcurrency),
		}
	}
	if req.Options.TimeoutMS < MinTimeoutMS || req.Options.TimeoutMS > MaxTimeoutMS {
		return &ValidationError{
			Field:   "options.timeout_ms",
			Message: fmt.Sprintf("doit être entre %d et %d", MinTimeoutMS, MaxTimeoutMS),
		}
	}
	return nil
}

func isHTTPURL(u string) bool {
	return len(u) > 7 &&
		(u[:7] == "http://" || (len(u) > 8 && u[:8] == "https://"))
}

func AggregateSummary(results []CheckResult, durationMS int64) Summary {
	counts := map[bool]int{} // true = up, false = down
	for _, r := range results {
		counts[r.OK]++
	}
	return Summary{
		Total:      len(results),
		Up:         counts[true],
		Down:       counts[false],
		DurationMS: durationMS,
	}
}
