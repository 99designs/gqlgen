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
	upgrader websocket.Upgrader
	recover  graphql.RecoverFunc
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
		if strings.Contains(r.Header.Get("Upgrade"), "websocket") {
			connectWs(exec, w, r, cfg.upgrader, cfg.recover)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		var reqParams params
		if r.Method == "GET" {
			reqParams.Query = r.URL.Query().Get("query")
			reqParams.OperationName = r.URL.Query().Get("operationName")

			if variables := r.URL.Query().Get("variables"); variables != "" {
				if err := json.Unmarshal([]byte(variables), &reqParams.Variables); err != nil {
					sendErrorf(w, http.StatusBadRequest, "variables could not be decoded")
					return
				}
			}
		} else {
			if err := json.NewDecoder(r.Body).Decode(&reqParams); err != nil {
				sendErrorf(w, http.StatusBadRequest, "json body could not be decoded: "+err.Error())
				return
			}
		}

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

		switch op.Type {
		case query.Query:
			b, err := json.Marshal(exec.Query(r.Context(), doc, reqParams.Variables, op, cfg.recover))
			if err != nil {
				panic(err)
			}
			w.Write(b)
		case query.Mutation:
			b, err := json.Marshal(exec.Mutation(r.Context(), doc, reqParams.Variables, op, cfg.recover))
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
