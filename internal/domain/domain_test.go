package domain_test

import (
	"errors"
	"testing"

	"github.com/MamadoubarryGLRSB/urlwatch/internal/domain"
)

func TestValidateRequest(t *testing.T) {
	cases := []struct {
		name    string
		req     domain.CheckRequest
		wantErr bool
		field   string
	}{
		{
			name: "valide nominal",
			req: domain.CheckRequest{
				URLs:    []string{"https://go.dev"},
				Options: domain.CheckOptions{Concurrency: 4, TimeoutMS: 2000},
			},
			wantErr: false,
		},
		{
			name: "urls vide",
			req: domain.CheckRequest{
				URLs:    []string{},
				Options: domain.CheckOptions{Concurrency: 4, TimeoutMS: 2000},
			},
			wantErr: true,
			field:   "urls",
		},
		{
			name: "url non http",
			req: domain.CheckRequest{
				URLs:    []string{"ftp://example.com"},
				Options: domain.CheckOptions{Concurrency: 4, TimeoutMS: 2000},
			},
			wantErr: true,
			field:   "urls",
		},
		{
			name: "concurrency trop faible",
			req: domain.CheckRequest{
				URLs:    []string{"https://go.dev"},
				Options: domain.CheckOptions{Concurrency: 0, TimeoutMS: 2000},
			},
			wantErr: true,
			field:   "options.concurrency",
		},
		{
			name: "concurrency trop élevée",
			req: domain.CheckRequest{
				URLs:    []string{"https://go.dev"},
				Options: domain.CheckOptions{Concurrency: 51, TimeoutMS: 2000},
			},
			wantErr: true,
			field:   "options.concurrency",
		},
		{
			name: "timeout trop court",
			req: domain.CheckRequest{
				URLs:    []string{"https://go.dev"},
				Options: domain.CheckOptions{Concurrency: 4, TimeoutMS: 50},
			},
			wantErr: true,
			field:   "options.timeout_ms",
		},
		{
			name: "timeout trop long",
			req: domain.CheckRequest{
				URLs:    []string{"https://go.dev"},
				Options: domain.CheckOptions{Concurrency: 4, TimeoutMS: 31000},
			},
			wantErr: true,
			field:   "options.timeout_ms",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := domain.ValidateRequest(&tc.req)
			if tc.wantErr && err == nil {
				t.Fatal("erreur attendue mais aucune reçue")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("erreur inattendue : %v", err)
			}
			if tc.wantErr && tc.field != "" {
				var ve *domain.ValidationError
				if !errors.As(err, &ve) {
					t.Fatalf("attendu *ValidationError, reçu %T", err)
				}
				if ve.Field != tc.field {
					t.Fatalf("champ attendu %q, reçu %q", tc.field, ve.Field)
				}
			}
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	opts := domain.CheckOptions{}
	domain.ApplyDefaults(&opts)

	if opts.Concurrency != domain.DefaultConcurrency {
		t.Errorf("concurrency attendue %d, reçu %d", domain.DefaultConcurrency, opts.Concurrency)
	}
	if opts.TimeoutMS != domain.DefaultTimeoutMS {
		t.Errorf("timeout attendu %d, reçu %d", domain.DefaultTimeoutMS, opts.TimeoutMS)
	}
}

func TestAggregateSummary(t *testing.T) {
	results := []domain.CheckResult{
		{URL: "https://a.com", OK: true},
		{URL: "https://b.com", OK: false},
		{URL: "https://c.com", OK: true},
	}
	summary := domain.AggregateSummary(results, 500)

	if summary.Total != 3 {
		t.Errorf("total attendu 3, reçu %d", summary.Total)
	}
	if summary.Up != 2 {
		t.Errorf("up attendu 2, reçu %d", summary.Up)
	}
	if summary.Down != 1 {
		t.Errorf("down attendu 1, reçu %d", summary.Down)
	}
	if summary.DurationMS != 500 {
		t.Errorf("duration_ms attendue 500, reçu %d", summary.DurationMS)
	}
}

func TestErrBatchNotFound(t *testing.T) {
	wrapped := errors.New("context: " + domain.ErrBatchNotFound.Error())
	err := errors.Join(domain.ErrBatchNotFound, wrapped)
	if !errors.Is(err, domain.ErrBatchNotFound) {
		t.Error("errors.Is devrait trouver ErrBatchNotFound dans le wrapping")
	}
}
