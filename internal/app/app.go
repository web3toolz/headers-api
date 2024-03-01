package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"headers-api/config"
	"headers-api/internal/ports/rest"
	"headers-api/pkg/dedup"
	"headers-api/pkg/subscriber"
	"log"
	"net/http"
)

type App struct {
	config         *config.Config
	rawHeaders     chan *types.Header
	cleanedHeaders chan *types.Header
	dedup          dedup.IDeduplication
	headersSubs    []subscriber.HeadersSubscriber
}

func createMux() *chi.Mux {
	router := chi.NewRouter()
	// Middleware
	router.Use(middleware.Recoverer)
	router.Use(middleware.Logger)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "OPTIONS", "HEAD"},
		MaxAge:         300,
	}))
	// Routes
	router.Get(`/`, rest.HealthCheckHandler)
	router.Get(`/ht`, rest.HealthCheckHandler)
	router.Get(`/ws`, rest.WebsocketHandler)
	router.Get(`/sse`, rest.SSEHandler)

	return router
}

func NewApp(config *config.Config) *App {
	return &App{
		config:         config,
		rawHeaders:     make(chan *types.Header),
		cleanedHeaders: make(chan *types.Header),
		dedup:          dedup.NewDeduplicationInstance(),
		headersSubs:    []subscriber.HeadersSubscriber{},
	}
}

func (app *App) createSubscriptions() error {
	ctx := context.Background()

	for _, sourceConf := range app.config.Sources {
		sub, err := subscriber.NewHeadersSubscriber(
			ctx,
			sourceConf.Url,
			app.rawHeaders,
			sourceConf.Timeout,
			sourceConf.PollingInterval,
		)

		if err != nil {
			log.Printf("Failed to create headers subscriber: %s\n", err)
			continue
		}

		err = sub.Subscribe()

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
	go func() {
		for {
			header := <-app.rawHeaders
			if !app.dedup.Deduplicate(header.Hash().Hex()) {
				app.cleanedHeaders <- header
			}
		}
	}()

	go func() {
		for {
			header := <-app.cleanedHeaders
			log.Printf("New header received: %d\n", header.Number.Uint64())
		}
	}()

	return nil
}

func (app *App) Run() error {
	err := app.runDeduplicationWorker()

	if err != nil {
		log.Fatal("Failed to start deduplication worker", err)
	}

	err = app.createSubscriptions()

	if err != nil {
		log.Fatal("Failed to create subscriptions", err)
	}

	mux := createMux()
	log.Printf("Starting server on port %d\n", app.config.Port)
	return http.ListenAndServe(fmt.Sprintf(":%d", app.config.Port), mux)
}
