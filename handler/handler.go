package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/gorilla/websocket"
)

func GraphQL(exec graphql.ExecutableSchema, options ...Option) http.HandlerFunc {
	var cfg Config

	for _, option := range options {
		option(&cfg)
	}

	srv := handler.New(exec)

	srv.AddTransport(transport.Websocket{
		Upgrader:              cfg.upgrader,
		InitFunc:              cfg.websocketInitFunc,
		KeepAlivePingInterval: cfg.connectionKeepAlivePingInterval,
	})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{
		MaxUploadSize: cfg.uploadMaxSize,
		MaxMemory:     cfg.uploadMaxMemory,
	})

	if cfg.cacheSize != 0 {
		srv.SetQueryCache(lru.New(cfg.cacheSize))
	}
	if cfg.recover != nil {
		srv.SetRecoverFunc(cfg.recover)
	}
	if cfg.errorPresenter != nil {
		srv.SetErrorPresenter(cfg.errorPresenter)
	}
	for _, hook := range cfg.fieldHooks {
		srv.AroundFields(hook)
	}
	for _, hook := range cfg.requestHooks {
		srv.AroundResponses(hook)
	}
	if cfg.complexityLimit != 0 {
		srv.Use(extension.ComplexityLimit(func(ctx context.Context, rc *graphql.RequestContext) int {
			return cfg.complexityLimit
		}))
	} else if cfg.complexityLimitFunc != nil {
		srv.Use(extension.ComplexityLimit(func(ctx context.Context, rc *graphql.RequestContext) int {
			return cfg.complexityLimitFunc(graphql.WithRequestContext(ctx, rc))
		}))
	}
	if !cfg.disableIntrospection {
		srv.Use(extension.Introspection{})
	}
	if cfg.apqCache != nil {
		srv.Use(extension.AutomaticPersistedQuery{Cache: apqAdapter{cfg.apqCache}})
	}
	return srv.ServeHTTP
}

type Config struct {
	cacheSize                       int
	upgrader                        websocket.Upgrader
	websocketInitFunc               transport.WebsocketInitFunc
	connectionKeepAlivePingInterval time.Duration
	recover                         graphql.RecoverFunc
	errorPresenter                  graphql.ErrorPresenterFunc
	fieldHooks                      []graphql.FieldMiddleware
	requestHooks                    []graphql.ResponseMiddleware
	complexityLimit                 int
	complexityLimitFunc             func(ctx context.Context) int
	disableIntrospection            bool
	uploadMaxMemory                 int64
	uploadMaxSize                   int64
	apqCache                        PersistedQueryCache
}

type Option func(cfg *Config)

func WebsocketUpgrader(upgrader websocket.Upgrader) Option {
	return func(cfg *Config) {
		cfg.upgrader = upgrader
	}
}

func RecoverFunc(recover graphql.RecoverFunc) Option {
	return func(cfg *Config) {
		cfg.recover = recover
	}
}

// ErrorPresenter transforms errors found while resolving into errors that will be returned to the user. It provides
// a good place to add any extra fields, like error.type, that might be desired by your frontend. Check the default
// implementation in graphql.DefaultErrorPresenter for an example.
func ErrorPresenter(f graphql.ErrorPresenterFunc) Option {
	return func(cfg *Config) {
		cfg.errorPresenter = f
	}
}

// IntrospectionEnabled = false will forbid clients from calling introspection endpoints. Can be useful in prod when you dont
// want clients introspecting the full schema.
func IntrospectionEnabled(enabled bool) Option {
	return func(cfg *Config) {
		cfg.disableIntrospection = !enabled
	}
}

// ComplexityLimit sets a maximum query complexity that is allowed to be executed.
// If a query is submitted that exceeds the limit, a 422 status code will be returned.
func ComplexityLimit(limit int) Option {
	return func(cfg *Config) {
		cfg.complexityLimit = limit
	}
}

// ComplexityLimitFunc allows you to define a function to dynamically set the maximum query complexity that is allowed
// to be executed.
// If a query is submitted that exceeds the limit, a 422 status code will be returned.
func ComplexityLimitFunc(complexityLimitFunc func(ctx context.Context) int) Option {
	return func(cfg *Config) {
		cfg.complexityLimitFunc = complexityLimitFunc
	}
}

// ResolverMiddleware allows you to define a function that will be called around every resolver,
// useful for logging.
func ResolverMiddleware(middleware graphql.FieldMiddleware) Option {
	return func(cfg *Config) {
		cfg.fieldHooks = append(cfg.fieldHooks, middleware)
	}
}

// RequestMiddleware allows you to define a function that will be called around the root request,
// after the query has been parsed. This is useful for logging
func RequestMiddleware(middleware graphql.ResponseMiddleware) Option {
	return func(cfg *Config) {
		cfg.requestHooks = append(cfg.requestHooks, middleware)
	}
}

// WebsocketInitFunc is called when the server receives connection init message from the client.
// This can be used to check initial payload to see whether to accept the websocket connection.
func WebsocketInitFunc(websocketInitFunc transport.WebsocketInitFunc) Option {
	return func(cfg *Config) {
		cfg.websocketInitFunc = websocketInitFunc
	}
}

// CacheSize sets the maximum size of the query cache.
// If size is less than or equal to 0, the cache is disabled.
func CacheSize(size int) Option {
	return func(cfg *Config) {
		cfg.cacheSize = size
	}
}

// UploadMaxSize sets the maximum number of bytes used to parse a request body
// as multipart/form-data.
func UploadMaxSize(size int64) Option {
	return func(cfg *Config) {
		cfg.uploadMaxSize = size
	}
}

// UploadMaxMemory sets the maximum number of bytes used to parse a request body
// as multipart/form-data in memory, with the remainder stored on disk in
// temporary files.
func UploadMaxMemory(size int64) Option {
	return func(cfg *Config) {
		cfg.uploadMaxMemory = size
	}
}

// WebsocketKeepAliveDuration allows you to reconfigure the keepalive behavior.
// By default, keepalive is enabled with a DefaultConnectionKeepAlivePingInterval
// duration. Set handler.connectionKeepAlivePingInterval = 0 to disable keepalive
// altogether.
func WebsocketKeepAliveDuration(duration time.Duration) Option {
	return func(cfg *Config) {
		cfg.connectionKeepAlivePingInterval = duration
	}
}

// Add cache that will hold queries for automatic persisted queries (APQ)
func EnablePersistedQueryCache(cache PersistedQueryCache) Option {
	return func(cfg *Config) {
		cfg.apqCache = cache
	}
}

func GetInitPayload(ctx context.Context) transport.InitPayload {
	return transport.GetInitPayload(ctx)
}

type apqAdapter struct {
	PersistedQueryCache
}

func (a apqAdapter) Get(key string) (value interface{}, ok bool) {
	return a.Get(key)
}
func (a apqAdapter) Add(key string, value interface{}) {
	a.Add(key, value)
}

type PersistedQueryCache interface {
	Add(ctx context.Context, hash string, query string)
	Get(ctx context.Context, hash string) (string, bool)
}
