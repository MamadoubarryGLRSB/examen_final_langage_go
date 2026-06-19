package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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

	// timeout global du lot
	batchTimeout := time.Duration(req.Options.TimeoutMS)*time.Millisecond*
		time.Duration(len(req.URLs)/req.Options.Concurrency+1) + 5*time.Second
	if batchTimeout > 60*time.Second {
		batchTimeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(r.Context(), batchTimeout)
	defer cancel()

	start := time.Now()
	results := pool.Run(ctx, h.checker, req.URLs, req.Options)
	durationMS := time.Since(start).Milliseconds()

	batch := domain.Batch{
		ID:        newBatchID(),
		CreatedAt: time.Now().UTC(),
		Results:   results,
		Summary:   domain.AggregateSummary(results, durationMS),
	}

	if err := h.store.Save(r.Context(), batch); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "impossible de sauvegarder le lot")
		return
	}

	setLogBatchID(r, batch.ID)
	writeJSON(w, http.StatusCreated, batch)
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

func Healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func newBatchID() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return "b_" + hex.EncodeToString(b) // ex: b_4f3c1a
}
