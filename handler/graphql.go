package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"strings"

	"github.com/gorilla/websocket"
	"github.com/vektah/gqlgen/graphql"
	"github.com/vektah/gqlgen/neelance/errors"
	"github.com/vektah/gqlgen/neelance/query"
	"github.com/vektah/gqlgen/neelance/validation"
)

type params struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

type Config struct {
	upgrader    websocket.Upgrader
	recover     graphql.RecoverFunc
	formatError func(error) string
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

func FormatErrorFunc(f func(error) string) Option {
	return func(cfg *Config) {
		cfg.formatError = f
	}
}

func GraphQL(exec graphql.ExecutableSchema, options ...Option) http.HandlerFunc {
	cfg := Config{
		recover: graphql.DefaultRecoverFunc,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}

	for _, option := range options {
		option(&cfg)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Allow", "OPTIONS, GET, POST")
			w.WriteHeader(http.StatusOK)
			return
		}

		if strings.Contains(r.Header.Get("Upgrade"), "websocket") {
			connectWs(exec, w, r, cfg.upgrader, cfg.recover)
			return
		}

		var reqParams params
		switch r.Method {
		case http.MethodGet:
			reqParams.Query = r.URL.Query().Get("query")
			reqParams.OperationName = r.URL.Query().Get("operationName")

			if variables := r.URL.Query().Get("variables"); variables != "" {
				if err := json.Unmarshal([]byte(variables), &reqParams.Variables); err != nil {
					sendErrorf(w, http.StatusBadRequest, "variables could not be decoded")
					return
				}
			}
		case http.MethodPost:
			if err := json.NewDecoder(r.Body).Decode(&reqParams); err != nil {
				sendErrorf(w, http.StatusBadRequest, "json body could not be decoded: "+err.Error())
				return
			}
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		doc, qErr := query.Parse(reqParams.Query)
		if qErr != nil {
			sendError(w, http.StatusUnprocessableEntity, qErr)
			return
		}

		errs := validation.Validate(exec.Schema(), doc)
		if len(errs) != 0 {
			sendError(w, http.StatusUnprocessableEntity, errs...)
			return
		}

		op, err := doc.GetOperation(reqParams.OperationName)
		if err != nil {
			sendErrorf(w, http.StatusUnprocessableEntity, err.Error())
			return
		}

		ctx := graphql.WithRequestContext(r.Context(), &graphql.RequestContext{
			Doc:       doc,
			Variables: reqParams.Variables,
			Recover:   cfg.recover,
			Builder: errors.Builder{
				ErrorMessageFn: cfg.formatError,
			},
		})

		switch op.Type {
		case query.Query:
			b, err := json.Marshal(exec.Query(ctx, op))
			if err != nil {
				panic(err)
			}
			w.Write(b)
		case query.Mutation:
			b, err := json.Marshal(exec.Mutation(ctx, op))
			if err != nil {
				panic(err)
			}
			w.Write(b)
		default:
			sendErrorf(w, http.StatusBadRequest, "unsupported operation type")
		}
	})
}

func sendError(w http.ResponseWriter, code int, errs ...*errors.QueryError) {
	w.WriteHeader(code)
	b, err := json.Marshal(&graphql.Response{Errors: errs})
	if err != nil {
		panic(err)
	}
	w.Write(b)
}

func sendErrorf(w http.ResponseWriter, code int, format string, args ...interface{}) {
	sendError(w, code, &errors.QueryError{Message: fmt.Sprintf(format, args...)})
}
