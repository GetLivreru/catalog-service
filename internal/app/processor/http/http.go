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

func NewHTTP(hHealth rhandler.Health, hCategory rhandler.Category, hProduct rhandler.Product, cfg section.ProcessorWebServer) *httpProc {
	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(handlerNotFound)
	vGenericRegHealthCheck(r, hHealth)
	rV1 := r.PathPrefix("/v1").Subrouter()
	v1RegCategoryHandler(rV1, hCategory)
	v1RegProductHandler(rV1, hProduct)

	_ = r.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		path, _ := route.GetPathTemplate()
		methods, _ := route.GetMethods()
		if path == "" || len(methods) == 0 {
			return nil
		}
		log.Printf("Route: %v %s", methods, path)
		return nil
	})

	p := httpProc{addr: fmt.Sprintf(":%d", cfg.ListenPort)}
	p.server.Addr = p.addr
	p.server.Handler = r

	return &p
}

func (p *httpProc) Serve() error {
	log.Printf("Starting HTTP server on %s", p.addr)
	return p.server.ListenAndServe()
}
