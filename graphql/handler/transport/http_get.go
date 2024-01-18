package transport

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/errcode"
)

// GET implements the GET side of the default HTTP transport
// defined in https://github.com/APIs-guru/graphql-over-http#get
type GET struct {
	// Map of all headers that are added to graphql response. If not
	// set, only one header: Content-Type: application/json will be set.
	ResponseHeaders map[string][]string
}

var _ graphql.Transport = GET{}

func (h GET) Supports(r *http.Request) bool {
	if r.Header.Get("Upgrade") != "" {
		return false
	}

	return r.Method == "GET"
}

func (h GET) Do(w http.ResponseWriter, r *http.Request, exec graphql.GraphExecutor) {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJsonError(w, err.Error())
		return
	}
	writeHeaders(w, h.ResponseHeaders)

	raw := &graphql.RawParams{
		Query:         query.Get("query"),
		OperationName: query.Get("operationName"),
		Headers:       r.Header,
	}
	raw.ReadTime.Start = graphql.Now()

	if variables := query.Get("variables"); variables != "" {
		if err := jsonDecode(strings.NewReader(variables), &raw.Variables); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			writeJsonError(w, "variables could not be decoded")
			return
		}
	}

	if extensions := query.Get("extensions"); extensions != "" {
		if err := jsonDecode(strings.NewReader(extensions), &raw.Extensions); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			writeJsonError(w, "extensions could not be decoded")
			return
		}
	}

	raw.ReadTime.End = graphql.Now()

	rc, gqlError := exec.CreateOperationContext(r.Context(), raw)
	if gqlError != nil {
		w.WriteHeader(statusFor(gqlError))
		resp := exec.DispatchError(graphql.WithOperationContext(r.Context(), rc), gqlError)
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
	writeJson(w, responses(ctx))
}

func jsonDecode(r io.Reader, val interface{}) error {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	return dec.Decode(val)
}

func statusFor(errs gqlerror.List) int {
	switch errcode.GetErrorKind(errs) {
	case errcode.KindProtocol:
		return http.StatusUnprocessableEntity
	default:
		return http.StatusOK
	}
}
