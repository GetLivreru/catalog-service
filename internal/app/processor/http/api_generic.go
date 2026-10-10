package rprocessor

import (
	"net/http"

	"github.com/gorilla/mux"

	rhandler "github.com/GetLivreru/catalog-service/internal/app/handler/http"
)

func vGenericRegHealthCheck(r *mux.Router, h rhandler.Health) {
	// TODO: зарегистрируйте GET /health
	// Используйте: reg(r, http.MethodGet, "/health", http.HandlerFunc(h.LastCheck))
	reg(r, http.MethodGet, "/health", http.HandlerFunc(h.LastCheck))
}

func handlerNotFound(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotFound)
}
