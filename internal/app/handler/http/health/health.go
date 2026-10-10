package rhealth

import (
	"net/http"

	rhandler "github.com/GetLivreru/catalog-service/internal/app/handler/http"
)

type handler struct{}

func NewHandler() rhandler.Health {
	return &handler{}
}

func (h *handler) LastCheck(w http.ResponseWriter, r *http.Request) {
	// TODO: реализуйте
	// Запишите "ok" в тело ответа
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
