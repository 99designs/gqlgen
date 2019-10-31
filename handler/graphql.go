package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/99designs/gqlgen/complexity"
	"github.com/99designs/gqlgen/graphql"
	"github.com/gorilla/websocket"
	lru "github.com/hashicorp/golang-lru"
	"github.com/vektah/gqlparser/ast"
	"github.com/vektah/gqlparser/gqlerror"
	"github.com/vektah/gqlparser/parser"
	"github.com/vektah/gqlparser/validator"
)

type params struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
	Extensions    *extensions            `json:"extensions"`
}

type extensions struct {
	PersistedQuery *persistedQuery `json:"persistedQuery"`
}

type persistedQuery struct {
	Sha256  string `json:"sha256Hash"`
	Version int64  `json:"version"`
}

type websocketInitFunc func(ctx context.Context, initPayload InitPayload) (context.Context, error)

type Config struct {
	cacheSize                       int
	upgrader                        websocket.Upgrader
	recover                         graphql.RecoverFunc
	errorPresenter                  graphql.ErrorPresenterFunc
	resolverHook                    graphql.FieldMiddleware
	tracer                          graphql.Tracer
	complexityLimit                 int
	complexityLimitFunc             graphql.ComplexityLimitFunc
	websocketInitFunc               websocketInitFunc
	disableIntrospection            bool
	connectionKeepAlivePingInterval time.Duration
}

func (c *Config) newRequestContext(ctx context.Context, es graphql.ExecutableSchema, doc *ast.QueryDocument, op *ast.OperationDefinition, operationName, query string, variables map[string]interface{}) (*graphql.RequestContext, error) {
	reqCtx := &graphql.RequestContext{
		Doc:                  doc,
		RawQuery:             query,
		Variables:            variables,
		OperationName:        operationName,
		DisableIntrospection: c.disableIntrospection,
		Recover:              c.recover,
		ResolverMiddleware:   c.resolverHook,
		ComplexityLimit:      c.complexityLimit,
	}
	if reqCtx.ComplexityLimit > 0 || c.complexityLimitFunc != nil {
		reqCtx.OperationComplexity = complexity.Calculate(es, op, variables)
	}
	err := reqCtx.Validate(ctx)
	if err != nil {
		return nil, err
	}

	return reqCtx, nil
}

type Option func(cfg *Config)

func WebsocketUpgrader(upgrader websocket.Upgrader) Option {
	return func(cfg *Config) {
		cfg.upgrader = upgrader
	}
}

// ResolverMiddleware allows you to define a function that will be called around every resolver,
// useful for logging.
func ResolverMiddleware(middleware graphql.FieldMiddleware) Option {
	return func(cfg *Config) {
		if cfg.resolverHook == nil {
			cfg.resolverHook = middleware
			return
		}

		lastResolve := cfg.resolverHook
		cfg.resolverHook = func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
			return lastResolve(ctx, func(ctx context.Context) (res interface{}, err error) {
				return middleware(ctx, next)
			})
		}
	}
}

// WebsocketInitFunc is called when the server receives connection init message from the client.
// This can be used to check initial payload to see whether to accept the websocket connection.
func WebsocketInitFunc(websocketInitFunc websocketInitFunc) Option {
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
// By default, keepalive is enabled with a DefaultConnectionKeepAlivePingInterval
// duration. Set handler.connectionKeepAlivePingInterval = 0 to disable keepalive
// altogether.
func WebsocketKeepAliveDuration(duration time.Duration) Option {
	return func(cfg *Config) {
		cfg.connectionKeepAlivePingInterval = duration
	}
}

const DefaultCacheSize = 1000
const DefaultConnectionKeepAlivePingInterval = 25 * time.Second

func GraphQL(exec graphql.ExecutableSchema, options ...Option) http.HandlerFunc {
	cfg := &Config{
		cacheSize:                       DefaultCacheSize,
		connectionKeepAlivePingInterval: DefaultConnectionKeepAlivePingInterval,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}

	for _, option := range options {
		option(cfg)
	}

	var cache *lru.Cache
	if cfg.cacheSize > 0 {
		var err error
		cache, err = lru.New(cfg.cacheSize)
		if err != nil {
			// An error is only returned for non-positive cache size
			// and we already checked for that.
			panic("unexpected error creating cache: " + err.Error())
		}
	}
	if cfg.tracer == nil {
		cfg.tracer = &graphql.NopTracer{}
	}

	handler := &graphqlHandler{
		cfg:   cfg,
		cache: cache,
		exec:  exec,
	}

	return handler.ServeHTTP
}

var _ http.Handler = (*graphqlHandler)(nil)

type graphqlHandler struct {
	cfg   *Config
	cache *lru.Cache
	exec  graphql.ExecutableSchema
}

func (gh *graphqlHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Allow", "OPTIONS, GET, POST")
		w.WriteHeader(http.StatusOK)
		return
	}

	if strings.Contains(r.Header.Get("Upgrade"), "websocket") {
		connectWs(gh.exec, w, r, gh.cfg, gh.cache)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	var reqParams params
	switch r.Method {
	case http.MethodGet:
		reqParams.Query = r.URL.Query().Get("query")
		reqParams.OperationName = r.URL.Query().Get("operationName")

		if variables := r.URL.Query().Get("variables"); variables != "" {
			if err := jsonDecode(strings.NewReader(variables), &reqParams.Variables); err != nil {
				sendErrorf(w, http.StatusBadRequest, "variables could not be decoded")
				return
			}
		}

		if extensions := r.URL.Query().Get("extensions"); extensions != "" {
			if err := jsonDecode(strings.NewReader(extensions), &reqParams.Extensions); err != nil {
				sendErrorf(w, http.StatusBadRequest, "extensions could not be decoded")
				return
			}
		}
	case http.MethodPost:
		mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			sendErrorf(w, http.StatusBadRequest, "error parsing request Content-Type")
			return
		}

		switch mediaType {
		case "application/json":
			if err := jsonDecode(r.Body, &reqParams); err != nil {
				sendErrorf(w, http.StatusBadRequest, "json body could not be decoded: "+err.Error())
				return
			}

		case "multipart/form-data":
			var closers []io.Closer
			var tmpFiles []string
			defer func() {
				for i := len(closers) - 1; 0 <= i; i-- {
					_ = closers[i].Close()
				}
				for _, tmpFile := range tmpFiles {
					_ = os.Remove(tmpFile)
				}
			}()
			if err := processMultipart(w, r, &reqParams, &closers, &tmpFiles, gh.cfg.uploadMaxSize, gh.cfg.uploadMaxMemory); err != nil {
				sendErrorf(w, http.StatusBadRequest, "multipart body could not be decoded: "+err.Error())
				return
			}
		default:
			sendErrorf(w, http.StatusBadRequest, "unsupported Content-Type: "+mediaType)
			return
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	if reqParams.Query == "" {
		sendErrorf(w, http.StatusUnprocessableEntity, "Must provide query string")
		return
	}

	var doc *ast.QueryDocument
	var cacheHit bool
	if gh.cache != nil {
		val, ok := gh.cache.Get(reqParams.Query)
		if ok {
			doc = val.(*ast.QueryDocument)
			cacheHit = true
		}
	}

	ctx, doc, gqlErr := gh.parseOperation(ctx, &parseOperationArgs{
		Query:     reqParams.Query,
		CachedDoc: doc,
	})
	if gqlErr != nil {
		sendError(w, http.StatusUnprocessableEntity, gqlErr)
		return
	}

	ctx, op, vars, listErr := gh.validateOperation(ctx, &validateOperationArgs{
		Doc:           doc,
		OperationName: reqParams.OperationName,
		CacheHit:      cacheHit,
		R:             r,
		Variables:     reqParams.Variables,
	})
	if len(listErr) != 0 {
		sendError(w, http.StatusUnprocessableEntity, listErr...)
		return
	}

	if gh.cache != nil && !cacheHit {
		gh.cache.Add(reqParams.Query, doc)
	}

	reqCtx, err := gh.cfg.newRequestContext(ctx, gh.exec, doc, op, reqParams.OperationName, reqParams.Query, vars)
	if err != nil {
		sendErrorf(w, http.StatusBadRequest, "invalid RequestContext was generated: %s", err.Error())
		return
	}
	ctx = graphql.WithRequestContext(ctx, reqCtx)

	defer func() {
		if err := recover(); err != nil {
			userErr := reqCtx.Recover(ctx, err)
			sendErrorf(w, http.StatusUnprocessableEntity, userErr.Error())
		}
	}()

	if gh.cfg.complexityLimitFunc != nil {
		reqCtx.ComplexityLimit = gh.cfg.complexityLimitFunc(ctx)
	}

	if reqCtx.ComplexityLimit > 0 && reqCtx.OperationComplexity > reqCtx.ComplexityLimit {
		sendErrorf(w, http.StatusUnprocessableEntity, "operation has complexity %d, which exceeds the limit of %d", reqCtx.OperationComplexity, reqCtx.ComplexityLimit)
		return
	}

	switch op.Operation {
	case ast.Query:
		b, err := json.Marshal(gh.exec.Query(ctx, op))
		if err != nil {
			panic(err)
		}
		w.Write(b)
	case ast.Mutation:
		b, err := json.Marshal(gh.exec.Mutation(ctx, op))
		if err != nil {
			panic(err)
		}
		w.Write(b)
	default:
		sendErrorf(w, http.StatusBadRequest, "unsupported operation type")
	}
}

type parseOperationArgs struct {
	Query     string
	CachedDoc *ast.QueryDocument
}

func (gh *graphqlHandler) parseOperation(ctx context.Context, args *parseOperationArgs) (context.Context, *ast.QueryDocument, *gqlerror.Error) {
	ctx = gh.cfg.tracer.StartOperationParsing(ctx)
	defer func() { gh.cfg.tracer.EndOperationParsing(ctx) }()

	if args.CachedDoc != nil {
		return ctx, args.CachedDoc, nil
	}

	doc, gqlErr := parser.ParseQuery(&ast.Source{Input: args.Query})
	if gqlErr != nil {
		return ctx, nil, gqlErr
	}

	return ctx, doc, nil
}

type validateOperationArgs struct {
	Doc           *ast.QueryDocument
	OperationName string
	CacheHit      bool
	R             *http.Request
	Variables     map[string]interface{}
}

func (gh *graphqlHandler) validateOperation(ctx context.Context, args *validateOperationArgs) (context.Context, *ast.OperationDefinition, map[string]interface{}, gqlerror.List) {
	ctx = gh.cfg.tracer.StartOperationValidation(ctx)
	defer func() { gh.cfg.tracer.EndOperationValidation(ctx) }()

	if !args.CacheHit {
		listErr := validator.Validate(gh.exec.Schema(), args.Doc)
		if len(listErr) != 0 {
			return ctx, nil, nil, listErr
		}
	}

	op := args.Doc.Operations.ForName(args.OperationName)
	if op == nil {
		return ctx, nil, nil, gqlerror.List{gqlerror.Errorf("operation %s not found", args.OperationName)}
	}

	if op.Operation != ast.Query && args.R.Method == http.MethodGet {
		return ctx, nil, nil, gqlerror.List{gqlerror.Errorf("GET requests only allow query operations")}
	}

	vars, err := validator.VariableValues(gh.exec.Schema(), op, args.Variables)
	if err != nil {
		return ctx, nil, nil, gqlerror.List{err}
	}

	return ctx, op, vars, nil
}

func jsonDecode(r io.Reader, val interface{}) error {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	return dec.Decode(val)
}

func sendError(w http.ResponseWriter, code int, errors ...*gqlerror.Error) {
	w.WriteHeader(code)
	b, err := json.Marshal(&graphql.Response{Errors: errors})
	if err != nil {
		panic(err)
	}
	w.Write(b)
}

func sendErrorf(w http.ResponseWriter, code int, format string, args ...interface{}) {
	sendError(w, code, &gqlerror.Error{Message: fmt.Sprintf(format, args...)})
}
