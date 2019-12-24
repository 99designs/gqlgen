package transport

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/ast"
)

// GET implements the GET side of the default HTTP transport
// defined in https://github.com/APIs-guru/graphql-over-http#get
type GET struct {
	StatusCodeFunc func(ctx context.Context, resp *graphql.Response) int
}

var _ graphql.Transport = GET{}

func (h GET) Supports(r *http.Request) bool {
	if r.Header.Get("Upgrade") != "" {
		return false
	}

	return r.Method == "GET"
}

func (h GET) Do(w http.ResponseWriter, r *http.Request, exec graphql.GraphExecutor) {
	raw := &graphql.RawParams{
		Query:         r.URL.Query().Get("query"),
		OperationName: r.URL.Query().Get("operationName"),
	}

	if variables := r.URL.Query().Get("variables"); variables != "" {
		if err := jsonDecode(strings.NewReader(variables), &raw.Variables); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			writeJsonError(w, "variables could not be decoded")
			return
		}
	}

	if extensions := r.URL.Query().Get("extensions"); extensions != "" {
		if err := jsonDecode(strings.NewReader(extensions), &raw.Extensions); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			writeJsonError(w, "extensions could not be decoded")
			return
		}
	}

	rc, err := exec.CreateOperationContext(r.Context(), raw)
	if err != nil {
		ctx := graphql.WithOperationContext(r.Context(), rc)
		resp := exec.DispatchError(ctx, err)
		h.writeStatusCode(ctx, w, resp)
		writeJson(w, resp)
		return
	}
	op := rc.Doc.Operations.ForName(rc.OperationName)
	if op.Operation != ast.Query {
		w.WriteHeader(http.StatusNotAcceptable)
		writeJsonError(w, "GET requests only allow query operations")
		return
	}

	responses, ctx := exec.DispatchOperation(r.Context(), rc)
	resp := responses(ctx)
	h.writeStatusCode(ctx, w, resp)
	writeJson(w, resp)
}

func (h GET) writeStatusCode(ctx context.Context, w http.ResponseWriter, resp *graphql.Response) {
	if h.StatusCodeFunc == nil {
		w.WriteHeader(httpStatusCode(resp))
	} else {
		w.WriteHeader(h.StatusCodeFunc(ctx, resp))
	}
}

func jsonDecode(r io.Reader, val interface{}) error {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	return dec.Decode(val)
}
