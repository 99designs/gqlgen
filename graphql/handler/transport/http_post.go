package transport

import (
	"encoding/json"
	"mime"
	"net/http"

	"github.com/99designs/gqlgen/graphql"
)

// POST implements the POST side of the default HTTP transport
// defined in https://github.com/APIs-guru/graphql-over-http#post
type POST struct{}

var _ graphql.Transport = POST{}

func (h POST) Supports(r *http.Request) bool {
	if r.Header.Get("Upgrade") != "" {
		return false
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return false
	}

	return r.Method == "POST" && mediaType == "application/json"
}

func (h POST) Do(w http.ResponseWriter, r *http.Request, exec graphql.GraphExecutor) {
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

	var params *graphql.RawParams
	if err := jsonDecode(r.Body, &params); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		write.Errorf("json body could not be decoded: " + err.Error())
		return
	}

	rc, err := exec.CreateRequestContext(r.Context(), params)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		write.GraphqlErr(err...)
		return
	}
	responses, ctx := exec.DispatchRequest(r.Context(), rc)
	write(graphql.StatusResolverError, responses(ctx))
}
