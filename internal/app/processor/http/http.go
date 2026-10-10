package rprocessor

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/GetLivreru/catalog-service/internal/app/config/section"
	rhandler "github.com/GetLivreru/catalog-service/internal/app/handler/http"
)

type httpProc struct {
	server http.Server
	addr   string
}

func NewHTTP(hHealth rhandler.Health, cfg section.ProcessorWebServer) *httpProc {
	// TODO: реализуйте
	//
	// Шаг 1: Создайте роутер
	// - r := mux.NewRouter()
	r := mux.NewRouter()
	//
	// Шаг 2: Зарегистрируйте NotFoundHandler
	// - r.NotFoundHandler = http.HandlerFunc(handlerNotFound)
	r.NotFoundHandler = http.HandlerFunc(handlerNotFound)
	//
	// Шаг 3: Зарегистрируйте health-check
	// - vGenericRegHealthCheck(r, hHealth)
	vGenericRegHealthCheck(r, hHealth)

	//
	// Шаг 4: Логирование маршрутов (опционально)
	// - Используйте r.Walk() для вывода зарегистрированных путей

	p := httpProc{addr: fmt.Sprintf(":%d", cfg.ListenPort)}
	p.server.Addr = p.addr
	p.server.Handler = r

	return &p
}

func (p *httpProc) Serve() error {
	log.Printf("Starting HTTP server on %s", p.addr)
	return p.server.ListenAndServe()
}
