package builder

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"github.com/GetLivreru/catalog-service/internal/app/config"
	rhandler "github.com/GetLivreru/catalog-service/internal/app/handler/http"
	hcategory "github.com/GetLivreru/catalog-service/internal/app/handler/http/category"
	rhealth "github.com/GetLivreru/catalog-service/internal/app/handler/http/health"
	hproduct "github.com/GetLivreru/catalog-service/internal/app/handler/http/product"
	"github.com/GetLivreru/catalog-service/internal/app/processor"
	rprocessor "github.com/GetLivreru/catalog-service/internal/app/processor/http"
	pprocessor "github.com/GetLivreru/catalog-service/internal/app/processor/other"
	"github.com/GetLivreru/catalog-service/internal/app/repository"
	pcategory "github.com/GetLivreru/catalog-service/internal/app/repository/category"
	rcpostgres "github.com/GetLivreru/catalog-service/internal/app/repository/conn/postgres"
	pproduct "github.com/GetLivreru/catalog-service/internal/app/repository/product"
	"github.com/GetLivreru/catalog-service/internal/app/service"
	scategory "github.com/GetLivreru/catalog-service/internal/app/service/category"
	sproduct "github.com/GetLivreru/catalog-service/internal/app/service/product"
)

type Builder struct {
	loggerMu        sync.Mutex
	cCtx            *cli.Context
	ctx             context.Context
	wg              sync.WaitGroup
	err             error
	cfg             config.Config
	connPostgres    *rcpostgres.Client
	categoryRepo    repository.Category
	productRepo     repository.Product
	chErrors        chan error
	cancelFunc      context.CancelFunc
	signals         chan os.Signal
	categoryService service.Category
	productService  service.Product
	healthHandler   rhandler.Health
	categoryHandler rhandler.Category
	productHandler  rhandler.Product
	processors      []processor.Processor
}

func NewBuilder(cCtx *cli.Context) *Builder {
	b := Builder{cCtx: cCtx, chErrors: make(chan error, 4096), signals: make(chan os.Signal, 1)}
	b.ctx, b.cancelFunc = context.WithCancel(context.Background())
	signal.Notify(b.signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go b.waitForSignal(b.signals, b.cancelFunc)
	go b.printErrors()
	b.healthHandler = rhealth.NewHandler()
	return &b
}

func (b *Builder) BuildConfig() {
	b.exec(func(b *Builder) { b.buildConfig() })
}

func (b *Builder) Run() {
	if b.ctx.Err() != nil {
		log.Info().Msg("Shutdown during initialization")
		b.cleanup()
		return
	}
	if b.err != nil {
		log.Fatal().Err(b.err).Msg("Failed to initialize application")
	}
	log.Info().Msg("Application initialized")
	defer log.Info().Msg("Application completed")
	defer b.cleanup()
	for _, proc := range b.processors {
		proc.StartAsync(b.ctx, &b.wg)
	}
	b.wg.Wait()
}

////////////////////////////////////////////////////////////////////////////////
///// REPOSITORY CONNECTIONS ////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

func (b *Builder) BuildRepoConnPostgres() {
	b.exec(func(b *Builder) {
		conn, err := rcpostgres.NewClient(b.ctx, b.cfg.Repository.Postgres)
		if err != nil {
			b.err = err
			return
		}
		b.connPostgres = conn
	})
}

func (b *Builder) BuildRepoConnMigrator() {
	if b.connPostgres == nil {
		return
	}
	b.exec(func(b *Builder) {
		b.processors = append(b.processors, pprocessor.NewMigrator(b.connPostgres))
	})
}

////////////////////////////////////////////////////////////////////////////////
///// REPOSITORIES //////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

func (b *Builder) BuildRepoCategory() {
	b.exec(func(b *Builder) {
		b.categoryRepo = pcategory.NewRepoFromPostgres(b.connPostgres)
	}, b.connPostgres)
}

func (b *Builder) BuildRepoProduct() {
	b.exec(func(b *Builder) {
		b.productRepo = pproduct.NewRepoFromPostgres(b.connPostgres)
	}, b.connPostgres)
}

////////////////////////////////////////////////////////////////////////////////
///// SERVICES /////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

func (b *Builder) BuildServiceCategory() {
	b.exec(func(b *Builder) {
		b.categoryService = scategory.NewService(b.categoryRepo, b.productRepo)
	}, b.categoryRepo, b.productRepo)
}

func (b *Builder) BuildServiceProduct() {
	b.exec(func(b *Builder) {
		b.productService = sproduct.NewService(b.productRepo, b.categoryRepo)
	}, b.productRepo, b.categoryRepo)
}

////////////////////////////////////////////////////////////////////////////////
///// HANDLERS /////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

func (b *Builder) BuildHandlerHttpCategory() {
	b.exec(func(b *Builder) {
		b.categoryHandler = hcategory.NewHandler(b.categoryService)
	}, b.categoryService)
}

func (b *Builder) BuildHandlerHttpProduct() {
	b.exec(func(b *Builder) {
		b.productHandler = hproduct.NewHandler(b.productService)
	}, b.productService)
}

////////////////////////////////////////////////////////////////////////////////
///// PROCESSORS ///////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

func (b *Builder) BuildProcHttp() {
	b.exec(func(b *Builder) {
		b.processors = append(b.processors, rprocessor.NewHTTP(
			b.healthHandler, b.categoryHandler, b.productHandler, b.cfg.Processor.WebServer,
		))
	}, b.healthHandler)
}

////////////////////////////////////////////////////////////////////////////////
///// PRIVATE //////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

func (b *Builder) cleanup() {
	signal.Stop(b.signals)
	b.cancelFunc()
	if b.connPostgres != nil {
		if err := b.connPostgres.GetRawBunDB().Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close PostgreSQL")
		}
	}
	close(b.chErrors)
}

func (b *Builder) buildConfig() {
	b.loggerMu.Lock()
	defer b.loggerMu.Unlock()
	config.Load(config.LoadArgs{
		Output:          b.cCtx.App.Writer,
		EnableSimpleLog: b.cCtx.Bool("no-json"),
	})
	b.cfg = config.Root
}

func (b *Builder) waitForSignal(sig chan os.Signal, cancelFunc func()) {
	select {
	case received := <-sig:
		b.loggerMu.Lock()
		log.Info().Str("signal", received.String()).Msg("Shutdown is requested")
		b.loggerMu.Unlock()
		cancelFunc()
	case <-b.ctx.Done():
	}
}

func (b *Builder) printErrors() {
	for err := range b.chErrors {
		b.loggerMu.Lock()
		log.Error().Err(err).Msg("Got new error")
		b.loggerMu.Unlock()
	}
}

func (b *Builder) exec(cb func(b *Builder), requiredArgs ...any) {
	if b.err != nil || b.ctx.Err() != nil {
		return
	}
	for i, requiredArg := range requiredArgs {
		rv := reflect.ValueOf(requiredArg)
		if !rv.IsValid() {
			b.err = fmt.Errorf("BUG: required argument #%d is nil (check dependencies)", i)
			return
		}
		if rv.Type().Kind() == reflect.Struct || !rv.IsZero() {
			continue
		}
		b.err = fmt.Errorf("BUG: required %s, but empty", rv.Type().String())
		return
	}
	cb(b)
}
