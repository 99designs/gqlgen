package transport

import (
	"encoding/json"
	"mime"
	"net/http"

	"github.com/99designs/gqlgen/graphql"
)

type JsonPostTransport struct{}

var _ graphql.Transport = JsonPostTransport{}

func (H JsonPostTransport) Supports(r *http.Request) bool {
	if r.Header.Get("Upgrade") != "" {
		return false
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return false
	}

	return r.Method == "POST" && mediaType == "application/json"
}

func (H JsonPostTransport) Do(w http.ResponseWriter, r *http.Request, handler graphql.Handler) {
	w.Header().Set("Content-Type", "application/json")

	write := graphql.Writer(func(status graphql.Status, response *graphql.Response) {
		switch status {
		case graphql.StatusOk, graphql.StatusResolverError:
			w.WriteHeader(http.StatusOK)
		case graphql.StatusParseError, graphql.StatusValidationError:
			w.WriteHeader(http.StatusUnprocessableEntity)
		}

		b, err := json.Marshal(response)
		if err != nil {
			panic(err)
		}
		w.Write(b)
	})

	var params rawParams
	if err := jsonDecode(r.Body, &params); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		write.Errorf("json body could not be decoded: " + err.Error())
		return
	}

	rc := newRequestContext()
	rc.RawQuery = params.Query
	rc.OperationName = params.OperationName
	rc.Variables = params.Variables
	rc.Extensions = params.Extensions

	ctx := graphql.WithRequestContext(r.Context(), rc)
	handler(ctx, write)
}
