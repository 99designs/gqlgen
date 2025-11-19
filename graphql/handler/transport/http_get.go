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
	// set, only one header: Content-Type: application/graphql-response+json will be set.
	ResponseHeaders map[string][]string
	// UseGrapQLResponseJsonByDefault determines whether to use 'application/graphql-response+json'
	// as the response content type
	// when the Accept header is empty or 'application/*' or '*/*'.
	UseGrapQLResponseJsonByDefault bool
}

var _ graphql.Transport = GET{}

func (h GET) Supports(r *http.Request) bool {
	if r.Header.Get("Upgrade") != "" {
		return false
	}

	return r.Method == http.MethodGet
}

func (h GET) Do(w http.ResponseWriter, r *http.Request, exec graphql.GraphExecutor) {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJsonError(w, err.Error())
		return
	}
	contentType := determineResponseContentType(
		h.ResponseHeaders,
		r,
		h.UseGrapQLResponseJsonByDefault,
	)
	responseHeaders := mergeHeaders(
		map[string][]string{
			"Content-Type": {contentType},
		},
		h.ResponseHeaders,
	)
	writeHeaders(w, responseHeaders)

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

	opCtx, gqlError := exec.CreateOperationContext(r.Context(), raw)
	if gqlError != nil {
		if contentType == acceptApplicationGraphqlResponseJson {
			w.WriteHeader(statusForGraphQLResponse(gqlError))
		} else {
			w.WriteHeader(statusFor(gqlError))
		}
		resp := exec.DispatchError(graphql.WithOperationContext(r.Context(), opCtx), gqlError)
		writeJson(w, resp)
		return
	}
	op := opCtx.Doc.Operations.ForName(opCtx.OperationName)
	if op.Operation != ast.Query {
		w.WriteHeader(http.StatusNotAcceptable)
		writeJsonError(w, "GET requests only allow query operations")
		return
	}

	responses, ctx := exec.DispatchOperation(r.Context(), opCtx)
	writeJson(w, responses(ctx))
}

func jsonDecode(r io.Reader, val any) error {
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

func statusForGraphQLResponse(errs gqlerror.List) int {
	// https://graphql.github.io/graphql-over-http/draft/#sec-application-graphql-response-json
	switch errcode.GetErrorKind(errs) {
	case errcode.KindProtocol:
		return http.StatusBadRequest
	default:
		return http.StatusOK
	}
}
