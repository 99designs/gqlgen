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
}

type Option func(cfg *Config)

func WebsocketUpgrader(upgrader websocket.Upgrader) Option {
	return func(cfg *Config) {
		cfg.upgrader = upgrader
	}
}

func GraphQL(exec graphql.ExecutableSchema, options ...Option) http.HandlerFunc {
	cfg := Config{
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
			connectWs(exec, w, r, cfg.upgrader)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		var params params
		if r.Method == "GET" {
			params.Query = r.URL.Query().Get("query")
			params.OperationName = r.URL.Query().Get("operationName")

			if variables := r.URL.Query().Get("variables"); variables != "" {
				if err := json.Unmarshal([]byte(variables), &params.Variables); err != nil {
					sendErrorf(w, http.StatusBadRequest, "variables could not be decoded")
					return
				}
			}
		} else {
			if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
				sendErrorf(w, http.StatusBadRequest, "json body could not be decoded: "+err.Error())
				return
			}
		}

		doc, qErr := query.Parse(params.Query)
		if qErr != nil {
			sendError(w, http.StatusUnprocessableEntity, qErr)
			return
		}

		errs := validation.Validate(exec.Schema(), doc)
		if len(errs) != 0 {
			sendError(w, http.StatusUnprocessableEntity, errs...)
			return
		}

		op, err := doc.GetOperation(params.OperationName)
		if err != nil {
			sendErrorf(w, http.StatusUnprocessableEntity, err.Error())
			return
		}

		switch op.Type {
		case query.Query:
			exec.Query(r.Context(), doc, params.Variables, op).MarshalGQL(w)
		case query.Mutation:
			exec.Mutation(r.Context(), doc, params.Variables, op).MarshalGQL(w)
		default:
			sendErrorf(w, http.StatusBadRequest, "unsupported operation type")
		}
	})
}

func sendError(w http.ResponseWriter, code int, errs ...*errors.QueryError) {
	w.WriteHeader(code)

	resp := &graphql.Response{
		Errors: errs,
	}
	resp.MarshalGQL(w)
}

func sendErrorf(w http.ResponseWriter, code int, format string, args ...interface{}) {
	sendError(w, code, &errors.QueryError{Message: fmt.Sprintf(format, args...)})
}
