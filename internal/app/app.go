package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"headers-api/config"
	"headers-api/internal/ports/rest"
	"headers-api/pkg/dedup"
	"headers-api/pkg/eventbus"
	"headers-api/pkg/subscriber"
	"log"
	"net/http"
	"sync"
)

type App struct {
	config         *config.Config
	mu             sync.Mutex
	rawHeaders     chan *types.Header
	cleanedHeaders chan *types.Header
	dedup          dedup.IDeduplication
	headersSubs    []subscriber.HeadersSubscriber
	bus            eventbus.IEventBus
	latestHeader   *types.Header
}

func NewApp(config *config.Config) *App {
	bus := eventbus.New()
	return &App{
		config:      config,
		mu:          sync.Mutex{},
		dedup:       dedup.NewDeduplicationInstance(),
		headersSubs: []subscriber.HeadersSubscriber{},
		bus:         bus,
	}
}

func (app *App) createHTTPRouter(
	subscribe func(string, func(*types.Header)) error,
	unsubscribe func(string),
	getLatestHeader func() *types.Header,
) *chi.Mux {
	router := chi.NewRouter()
	// Middleware
	router.Use(middleware.Recoverer)
	router.Use(middleware.Logger)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "OPTIONS", "HEAD"},
		MaxAge:         300,
	}))
	//Routes
	router.Get(`/`, rest.HealthCheckHandler)
	router.Get(`/ht`, rest.HealthCheckHandler)
	router.Get(`/latest`, rest.HTTPLatestHeaderHandler(getLatestHeader))
	router.Get(`/ws`, rest.WebsocketHandler(subscribe, unsubscribe))
	router.Get(`/sse`, rest.SSEHandler(subscribe, unsubscribe))

	return router
}

func (app *App) createSubscriptions() error {
	ctx := context.Background()

	publishRawHeadersFunc := func(header *types.Header) {
		app.bus.Publish("headers:raw", header)
	}

	for _, sourceConf := range app.config.Sources {
		sub, err := subscriber.NewHeadersSubscriber(ctx, sourceConf.Url, publishRawHeadersFunc)

		if err != nil {
			log.Printf("Failed to create headers subscriber: %s\n", err)
			continue
		}

		err = sub.Subscribe(sourceConf.Timeout, sourceConf.PollingInterval)

		log.Printf("Subscribed to headers from %s\n", sourceConf.Url)

		if err != nil {
			log.Printf("Failed to subscribe to headers: %s\n", err)
			continue
		}
		app.headersSubs = append(app.headersSubs, *sub)
	}

	if len(app.headersSubs) == 0 {
		return errors.New("no headers sources configured")
	}
	return nil
}

func (app *App) runDeduplicationWorker() error {
	id := uuid.NewString()
	err := app.bus.Subscribe("headers:raw", id, func(header *types.Header) {
		if !app.dedup.Deduplicate(header.Hash().Hex()) {
			app.mu.Lock()
			app.bus.Publish("headers:cleaned", header)
			app.latestHeader = header
			app.mu.Unlock()
		}
	})
	return err
}

func (app *App) Run() error {
	subscribeFunc := func(id string, handler func(*types.Header)) error {
		return app.bus.Subscribe("headers:cleaned", id, handler)
	}
	unsubscribeFunc := func(id string) {
		app.bus.Unsubscribe("headers:cleaned", id)
	}
	getLatestHeader := func() *types.Header {
		app.mu.Lock()
		defer app.mu.Unlock()
		return app.latestHeader
	}

	err := app.runDeduplicationWorker()
	if err != nil {
		log.Fatal("Failed to start deduplication worker", err)
	}

	err = app.createSubscriptions()
	if err != nil {
		log.Fatal("Failed to create subscriptions", err)
	}

	mux := app.createHTTPRouter(subscribeFunc, unsubscribeFunc, getLatestHeader)
	log.Printf("Starting server on port %d\n", app.config.Port)
	return http.ListenAndServe(fmt.Sprintf(":%d", app.config.Port), mux)
}
