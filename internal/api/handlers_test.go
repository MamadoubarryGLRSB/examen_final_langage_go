package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MamadoubarryGLRSB/urlwatch/internal/api"
	"github.com/MamadoubarryGLRSB/urlwatch/internal/checker"
	"github.com/MamadoubarryGLRSB/urlwatch/internal/domain"
	"github.com/MamadoubarryGLRSB/urlwatch/internal/store"
)

func newTestHandler(responses map[string]domain.CheckResult) *api.Handler {
	mock := checker.NewMock(responses)
	st := store.NewMemory()
	return api.NewHandler(mock, st)
}

func TestPostChecksSuccess(t *testing.T) {
	h := newTestHandler(map[string]domain.CheckResult{
		"https://go.dev": {OK: true, StatusCode: 200, LatencyMS: 120},
	})

	body := `{"urls":["https://go.dev"],"options":{"concurrency":2,"timeout_ms":1000}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/checks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.PostChecks(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("attendu 201, reçu %d : %s", rr.Code, rr.Body.String())
	}

	var batch domain.Batch
	if err := json.NewDecoder(rr.Body).Decode(&batch); err != nil {
		t.Fatalf("réponse non décodable : %v", err)
	}
	if batch.ID == "" {
		t.Error("batch_id ne doit pas être vide")
	}
	if batch.Summary.Total != 1 {
		t.Errorf("total attendu 1, reçu %d", batch.Summary.Total)
	}
	if batch.Summary.Up != 1 {
		t.Errorf("up attendu 1, reçu %d", batch.Summary.Up)
	}
}

func TestPostChecksInvalidBody(t *testing.T) {
	h := newTestHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/checks", bytes.NewBufferString("{bad json"))
	rr := httptest.NewRecorder()

	h.PostChecks(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("attendu 400, reçu %d", rr.Code)
	}
}

func TestPostChecksValidation(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"urls vide", `{"urls":[],"options":{"concurrency":2,"timeout_ms":1000}}`},
		{"url non http", `{"urls":["ftp://bad.com"],"options":{"concurrency":2,"timeout_ms":1000}}`},
		{"concurrency trop élevée", `{"urls":["https://go.dev"],"options":{"concurrency":99,"timeout_ms":1000}}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestHandler(nil)
			req := httptest.NewRequest(http.MethodPost, "/v1/checks", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			h.PostChecks(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("attendu 400, reçu %d : %s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestGetCheckFound(t *testing.T) {
	mock := checker.NewMock(map[string]domain.CheckResult{
		"https://go.dev": {OK: true, StatusCode: 200},
	})
	st := store.NewMemory()
	h := api.NewHandler(mock, st)

	postBody := `{"urls":["https://go.dev"],"options":{"concurrency":1,"timeout_ms":1000}}`
	postReq := httptest.NewRequest(http.MethodPost, "/v1/checks", bytes.NewBufferString(postBody))
	postReq.Header.Set("Content-Type", "application/json")
	postRR := httptest.NewRecorder()
	h.PostChecks(postRR, postReq)

	var created domain.Batch
	_ = json.NewDecoder(postRR.Body).Decode(&created)

	getReq := httptest.NewRequest(http.MethodGet, "/v1/checks/"+created.ID, nil)
	getReq.SetPathValue("id", created.ID)
	getRR := httptest.NewRecorder()
	h.GetCheck(getRR, getReq)

	if getRR.Code != http.StatusOK {
		t.Fatalf("attendu 200, reçu %d : %s", getRR.Code, getRR.Body.String())
	}
}

func TestGetCheckNotFound(t *testing.T) {
	h := newTestHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/checks/b_inconnu", nil)
	req.SetPathValue("id", "b_inconnu")
	rr := httptest.NewRecorder()

	h.GetCheck(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("attendu 404, reçu %d", rr.Code)
	}
}

func TestPostChecksAsync(t *testing.T) {
	h := newTestHandler(map[string]domain.CheckResult{
		"https://go.dev": {OK: true, StatusCode: 200, LatencyMS: 10},
	})

	body := `{"urls":["https://go.dev"],"options":{"concurrency":1,"timeout_ms":1000}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/checks?async=true", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.PostChecks(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("attendu 202, reçu %d", rr.Code)
	}

	var resp struct {
		BatchID string `json:"batch_id"`
		Status  string `json:"status"`
	}
	_ = json.NewDecoder(rr.Body).Decode(&resp)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		getReq := httptest.NewRequest(http.MethodGet, "/v1/checks/"+resp.BatchID, nil)
		getReq.SetPathValue("id", resp.BatchID)
		getRR := httptest.NewRecorder()
		h.GetCheck(getRR, getReq)

		var batch domain.Batch
		_ = json.NewDecoder(getRR.Body).Decode(&batch)
		if batch.Status == domain.StatusDone {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("le lot async n'est pas passé à done")
}

func TestListChecks(t *testing.T) {
	h := newTestHandler(map[string]domain.CheckResult{
		"https://go.dev": {OK: true, StatusCode: 200},
	})

	body := `{"urls":["https://go.dev"],"options":{"concurrency":1,"timeout_ms":1000}}`
	postReq := httptest.NewRequest(http.MethodPost, "/v1/checks", bytes.NewBufferString(body))
	postReq.Header.Set("Content-Type", "application/json")
	postRR := httptest.NewRecorder()
	h.PostChecks(postRR, postReq)

	listReq := httptest.NewRequest(http.MethodGet, "/v1/checks?page=1&limit=10", nil)
	listRR := httptest.NewRecorder()
	h.ListChecks(listRR, listReq)

	if listRR.Code != http.StatusOK {
		t.Fatalf("attendu 200, reçu %d", listRR.Code)
	}

	var result domain.ListResult
	if err := json.NewDecoder(listRR.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Total < 1 {
		t.Errorf("total attendu >= 1, reçu %d", result.Total)
	}
}

func TestHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	api.Healthz(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("attendu 200, reçu %d", rr.Code)
	}
}

type mockStore struct{}

func (m *mockStore) Save(_ context.Context, _ domain.Batch) error { return nil }
func (m *mockStore) Get(_ context.Context, _ string) (domain.Batch, error) {
	return domain.Batch{}, domain.ErrBatchNotFound
}
func (m *mockStore) List(_ context.Context, _ domain.ListParams) (domain.ListResult, error) {
	return domain.ListResult{Items: []domain.Batch{}}, nil
}
