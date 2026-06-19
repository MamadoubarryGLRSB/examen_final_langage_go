package api

import (
	"log/slog"
	"net/http"
)

func NewRouter(h *Handler, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /v1/checks", h.PostChecks)
	mux.HandleFunc("GET /v1/checks/{id}", h.GetCheck)
	mux.HandleFunc("GET /healthz", Healthz)

	// recovery en dernier pour attraper les panics
	var handler http.Handler = mux
	handler = LoggingMiddleware(logger, handler)
	handler = RecoveryMiddleware(logger, handler)

	return handler
}
