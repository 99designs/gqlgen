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

func (H JsonPostTransport) Do(w http.ResponseWriter, r *http.Request) (*graphql.RequestContext, graphql.Writer) {
	w.Header().Set("Content-Type", "application/json")

	write := graphql.Writer(func(response *graphql.Response) {
		b, err := json.Marshal(response)
		if err != nil {
			panic(err)
		}
		w.Write(b)
	})

	var params struct {
		Query         string                 `json:"query"`
		OperationName string                 `json:"operationName"`
		Variables     map[string]interface{} `json:"variables"`
		Extensions    map[string]interface{} `json:"extensions"`
	}
	if err := jsonDecode(r.Body, &params); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		write.Errorf("json body could not be decoded: " + err.Error())
		return nil, nil
	}

	reqParams := newRequestContext()
	reqParams.RawQuery = params.Query
	reqParams.OperationName = params.OperationName
	reqParams.Variables = params.Variables
	reqParams.Extensions = params.Extensions

	return reqParams, write
}
