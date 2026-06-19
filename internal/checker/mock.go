package checker

import (
	"context"

	"github.com/MamadoubarryGLRSB/urlwatch/internal/domain"
)

// utilisé dans les tests, pas d'appel réseau
type MockChecker struct {
	Responses map[string]domain.CheckResult
	Default   domain.CheckResult
}

func NewMock(responses map[string]domain.CheckResult) *MockChecker {
	return &MockChecker{Responses: responses}
}

func (m *MockChecker) Check(ctx context.Context, url string) domain.CheckResult {
	select {
	case <-ctx.Done():
		return domain.CheckResult{
			URL:   url,
			OK:    false,
			Error: ctx.Err().Error(),
		}
	default:
	}

	if r, ok := m.Responses[url]; ok {
		r.URL = url
		return r
	}
	res := m.Default
	res.URL = url
	return res
}
