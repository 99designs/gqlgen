package relay

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/vektah/graphql-go/errors"
)

type Resolver func(ctx context.Context, document string, operationName string, variables map[string]interface{}, w io.Writer) []*errors.QueryError

func Handler(resolver Resolver) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var params struct {
			Query         string                 `json:"query"`
			OperationName string                 `json:"operationName"`
			Variables     map[string]interface{} `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		errs := resolver(r.Context(), params.Query, params.OperationName, params.Variables, w)
		if errs != nil {
			w.WriteHeader(http.StatusBadRequest)

			b, err := json.Marshal(struct {
				Errors []*errors.QueryError `json:"errors"`
			}{Errors: errs})
			if err != nil {
				panic(err)
			}
			w.Write(b)
		}
	})
}
