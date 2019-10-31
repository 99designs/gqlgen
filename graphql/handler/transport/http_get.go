package transport

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/ast"
)

// GET implements the GET side of the default HTTP transport
// defined in https://github.com/APIs-guru/graphql-over-http#get
type GET struct{}

var _ graphql.Transport = GET{}

func (H GET) Supports(r *http.Request) bool {
	if r.Header.Get("Upgrade") != "" {
		return false
	}

	return r.Method == "GET"
}

func (H GET) Do(w http.ResponseWriter, r *http.Request, exec graphql.GraphExecutor) {
	raw := &graphql.RawParams{
		Query:         r.URL.Query().Get("query"),
		OperationName: r.URL.Query().Get("operationName"),
	}

	writer := graphql.Writer(func(status graphql.Status, response *graphql.Response) {
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

	if variables := r.URL.Query().Get("variables"); variables != "" {
		if err := jsonDecode(strings.NewReader(variables), &raw.Variables); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			writer.Errorf("variables could not be decoded")
			return
		}
	}

	if extensions := r.URL.Query().Get("extensions"); extensions != "" {
		if err := jsonDecode(strings.NewReader(extensions), &raw.Extensions); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			writer.Errorf("extensions could not be decoded")
			return
		}
	}

	rc, err := exec.CreateRequestContext(r.Context(), raw)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		writer.GraphqlErr(err...)
		return
	}
	op := rc.Doc.Operations.ForName(rc.OperationName)
	if op.Operation != ast.Query {
		w.WriteHeader(http.StatusNotAcceptable)
		writer.Errorf("GET requests only allow query operations")
		return
	}
	ctx := graphql.WithRequestContext(r.Context(), rc)
	exec.DispatchRequest(ctx, writer)
}

func jsonDecode(r io.Reader, val interface{}) error {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	return dec.Decode(val)
}
