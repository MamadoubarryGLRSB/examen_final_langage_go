package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/MamadoubarryGLRSB/urlwatch/internal/domain"
	"github.com/MamadoubarryGLRSB/urlwatch/internal/pool"
)

type Handler struct {
	checker domain.Checker
	store   domain.Store
}

func NewHandler(c domain.Checker, s domain.Store) *Handler {
	return &Handler{checker: c, store: s}
}

func (h *Handler) PostChecks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "méthode non autorisée")
		return
	}

	var req domain.CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "corps JSON invalide : "+err.Error())
		return
	}

	domain.ApplyDefaults(&req.Options)

	if err := domain.ValidateRequest(&req); err != nil {
		var ve *domain.ValidationError
		if errors.As(err, &ve) {
			writeError(w, http.StatusBadRequest, "invalid_request", ve.Error())
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if r.URL.Query().Get("async") == "true" {
		h.postChecksAsync(w, r, req)
		return
	}

	batch, err := h.runBatch(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "impossible de traiter le lot")
		return
	}

	if err := h.store.Save(r.Context(), batch); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "impossible de sauvegarder le lot")
		return
	}

	setLogBatchID(r, batch.ID)
	writeJSON(w, http.StatusCreated, batch)
}

func (h *Handler) postChecksAsync(w http.ResponseWriter, r *http.Request, req domain.CheckRequest) {
	batch := domain.Batch{
		ID:        newBatchID(),
		CreatedAt: time.Now().UTC(),
		Status:    domain.StatusPending,
		Results:   []domain.CheckResult{},
	}

	if err := h.store.Save(r.Context(), batch); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "impossible de sauvegarder le lot")
		return
	}

	setLogBatchID(r, batch.ID)

	// le traitement continue en arrière-plan
	go func() {
		done, err := h.runBatch(context.Background(), req)
		if err != nil {
			return
		}
		done.ID = batch.ID
		done.CreatedAt = batch.CreatedAt
		done.Status = domain.StatusDone
		_ = h.store.Save(context.Background(), done)
	}()

	writeJSON(w, http.StatusAccepted, map[string]any{
		"batch_id":   batch.ID,
		"status":     domain.StatusPending,
		"created_at": batch.CreatedAt,
	})
}

func (h *Handler) runBatch(ctx context.Context, req domain.CheckRequest) (domain.Batch, error) {
	batchTimeout := time.Duration(req.Options.TimeoutMS)*time.Millisecond*
		time.Duration(len(req.URLs)/req.Options.Concurrency+1) + 5*time.Second
	if batchTimeout > 60*time.Second {
		batchTimeout = 60 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, batchTimeout)
	defer cancel()

	start := time.Now()
	results := pool.Run(runCtx, h.checker, req.URLs, req.Options)
	durationMS := time.Since(start).Milliseconds()

	return domain.Batch{
		ID:        newBatchID(),
		CreatedAt: time.Now().UTC(),
		Results:   results,
		Summary:   domain.AggregateSummary(results, durationMS),
	}, nil
}

func (h *Handler) GetCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "méthode non autorisée")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "identifiant manquant")
		return
	}
	setLogBatchID(r, id)

	batch, err := h.store.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrBatchNotFound) {
			writeError(w, http.StatusNotFound, "batch_not_found",
				fmt.Sprintf("aucun lot avec l'id %s", id))
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "erreur interne")
		return
	}

	writeJSON(w, http.StatusOK, batch)
}

func (h *Handler) ListChecks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "méthode non autorisée")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 {
		limit = domain.DefaultListLimit
	}
	if limit > domain.MaxListLimit {
		limit = domain.MaxListLimit
	}

	result, err := h.store.List(r.Context(), domain.ListParams{
		Status: r.URL.Query().Get("status"),
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "erreur interne")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func Healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func newBatchID() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return "b_" + hex.EncodeToString(b) // ex: b_4f3c1a
}
