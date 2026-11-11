package rprocessor

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"

	"github.com/GetLivreru/catalog-service/internal/app/config/section"
	rhandler "github.com/GetLivreru/catalog-service/internal/app/handler/http"
	"github.com/GetLivreru/catalog-service/internal/app/processor"
	"github.com/GetLivreru/catalog-service/internal/app/util"
	"github.com/GetLivreru/catalog-service/internal/pkg/http/httph"
	"github.com/GetLivreru/catalog-service/internal/pkg/http/mzerolog"
)

type httpProc struct {
	server http.Server
	addr   string
}

func NewHTTP(hHealth rhandler.Health, hCategory rhandler.Category, hProduct rhandler.Product, cfg section.ProcessorWebServer) processor.Processor {
	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(handlerNotFound)
	r.Use(httph.NewErrorMiddleware(), mzerolog.NewMiddleware(mzerolog.WithSkipper(util.IsFilteredHttpRoute)))
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
		log.Info().Strs("methods", methods).Str("path", path).Msg("Route registered")
		return nil
	})

	p := httpProc{addr: fmt.Sprintf(":%d", cfg.ListenPort)}
	p.server.Handler = r

	return &p
}

func (p *httpProc) StartAsync(ctx context.Context, wg *sync.WaitGroup) {
	if ctx.Err() != nil {
		return
	}
	lc := net.ListenConfig{}
	listener, err := lc.Listen(ctx, "tcp", p.addr)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		log.Fatal().Err(err).Msg("Failed to start listening TCP addr")
	}
	log.Info().Str("listen_addr", p.addr).Msg("Listening of TCP addr for HTTP server has been started")
	go p.serve(listener)
	processor.WatchForShutdown(ctx, wg, processor.CloserFunc(listener.Close))
	processor.WatchForShutdown(ctx, wg, processor.NewCloserContextFunc(p.server.Shutdown, context.Background(), 5*time.Second))
}

func (p *httpProc) serve(listener net.Listener) {
	_ = p.server.Serve(listener)
}
