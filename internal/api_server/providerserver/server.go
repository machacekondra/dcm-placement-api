package catalogserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	api "github.com/dcm-project/dcm-placement-api/api/v1alpha1/provider"
	server "github.com/dcm-project/dcm-placement-api/internal/api/server/catalog"
	"github.com/dcm-project/dcm-placement-api/internal/config"
	handlers "github.com/dcm-project/dcm-placement-api/internal/handlers/v1alpha1"
	"github.com/dcm-project/dcm-placement-api/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	oapimiddleware "github.com/oapi-codegen/nethttp-middleware"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
)

const (
	gracefulShutdownTimeout = 5 * time.Second
)

type Server struct {
	cfg      *config.Config
	store    store.Store
	listener net.Listener
}

// New returns a new instance of a migration-planner server.
func New(
	cfg *config.Config,
	store store.Store,
	listener net.Listener,
) *Server {
	return &Server{
		cfg:      cfg,
		store:    store,
		listener: listener,
	}
}

func oapiErrorHandler(w http.ResponseWriter, message string, statusCode int) {
	http.Error(w, fmt.Sprintf("API Error: %s", message), statusCode)
}

func (s *Server) Run(ctx context.Context) error {
	zap.S().Named("provider_server").Info("Initializing provider server")
	swagger, err := api.GetSwagger()
	if err != nil {
		return fmt.Errorf("failed to load swagger spec: %w", err)
	}
	// Skip server name validation
	swagger.Servers = nil

	oapiOpts := oapimiddleware.Options{
		ErrorHandlerWithOpts: func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request, opts oapimiddleware.ErrorHandlerOpts) {
			zap.S().Named("provider_server").Warnf("OpenAPI validation error: %v (status: %d, route: %v)", err, opts.StatusCode, opts.MatchedRoute)
			oapiErrorHandler(w, err.Error(), opts.StatusCode)
		},
		ErrorHandler: oapiErrorHandler,
	}
	router := chi.NewRouter()

	router.Use(
		middleware.RequestID,
		middleware.Recoverer,
	)

	// Add Swagger UI endpoints BEFORE OpenAPI validation middleware
	router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger.json"),
	))
	router.Get("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		swaggerJSON, err := json.Marshal(swagger)
		if err != nil {
			http.Error(w, "Failed to marshal swagger spec", http.StatusInternalServerError)
			return
		}
		_, err = w.Write(swaggerJSON)
		if err != nil {
			return
		}
	})

	h := handlers.NewResourceHandler(s.store)

	router.Group(func(r chi.Router) {
		// 1. CORS must be FIRST
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: false,
			MaxAge:           300,
		}))

		// 2. Store request in context for content negotiation
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := context.WithValue(r.Context(), "http_request", r)
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		})

		// 3. OpenAPI validator AFTER CORS
		r.Use(oapimiddleware.OapiRequestValidatorWithOptions(swagger, &oapiOpts))

		// 4. Register all routes LAST
		server.HandlerFromMux(server.NewStrictHandler(h, nil), r)
	})

	srv := http.Server{Addr: s.cfg.Provider.Address, Handler: router}

	go func() {
		<-ctx.Done()
		zap.S().Named("provider_server").Infof("Shutdown signal received: %s", ctx.Err())
		ctxTimeout, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
		defer cancel()

		srv.SetKeepAlivesEnabled(false)
		_ = srv.Shutdown(ctxTimeout)
	}()

	zap.S().Named("provider_server").Infof("Listening on %s...", s.listener.Addr().String())
	if err := srv.Serve(s.listener); err != nil && !errors.Is(err, net.ErrClosed) {
		return err
	}

	return nil
}
