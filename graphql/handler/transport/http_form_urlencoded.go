package transport

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/99designs/gqlgen/graphql"
)

// FORM implements the application/x-www-form-urlencoded side of the default HTTP transport
type UrlEncodedForm struct {
	// Map of all headers that are added to graphql response. If not
	// set, only one header: Content-Type: application/json will be set.
	ResponseHeaders map[string][]string
}

var _ graphql.Transport = UrlEncodedForm{}

func (h UrlEncodedForm) Supports(r *http.Request) bool {
	if r.Header.Get("Upgrade") != "" {
		return false
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return false
	}

	return r.Method == http.MethodPost && mediaType == "application/x-www-form-urlencoded"
}

func (h UrlEncodedForm) Do(w http.ResponseWriter, r *http.Request, exec graphql.GraphExecutor) {
	ctx := r.Context()
	writeHeaders(w, h.ResponseHeaders)
	start := graphql.Now()

	bodyString, err := getRequestBody(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		gqlErr := gqlerror.Errorf("could not get form body: %+v", err)
		resp := exec.DispatchError(ctx, gqlerror.List{gqlErr})
		writeJson(w, resp)
		return
	}

	params, err := h.parseBody(bodyString)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		gqlErr := gqlerror.Errorf("could not cleanup body: %+v", err)
		resp := exec.DispatchError(ctx, gqlerror.List{gqlErr})
		writeJson(w, resp)
		return
	}

	params.Headers = r.Header
	params.ReadTime = graphql.TraceTiming{
		Start: start,
		End:   graphql.Now(),
	}

	rc, opErr := exec.CreateOperationContext(ctx, params)
	if opErr != nil {
		w.WriteHeader(statusFor(opErr))
		resp := exec.DispatchError(graphql.WithOperationContext(ctx, rc), opErr)
		writeJson(w, resp)
		return
	}

	var responses graphql.ResponseHandler
	responses, ctx = exec.DispatchOperation(ctx, rc)
	writeJson(w, responses(ctx))
}

func (h UrlEncodedForm) parseBody(bodyString string) (*graphql.RawParams, error) {
	if strings.Contains(bodyString, "\"query\":") {
		// body is json
		return h.parseJson(bodyString)
	}

	// Bodies that do not decode to a form with a query field (raw GraphQL
	// documents, or malformed escapes) fall back to the plain text handling.
	if form, err := url.ParseQuery(bodyString); err == nil && form.Has("query") {
		// body is urlencoded
		return h.parseForm(form)
	}

	// body is plain text
	params := &graphql.RawParams{}
	params.Query = strings.TrimPrefix(bodyString, "query=")

	return params, nil
}

// parseForm reads the request parameters from the query, operationName,
// variables and extensions form fields, decoding the last two as JSON.
func (h UrlEncodedForm) parseForm(form url.Values) (*graphql.RawParams, error) {
	params := &graphql.RawParams{
		Query:         form.Get("query"),
		OperationName: form.Get("operationName"),
	}

	if variables := form.Get("variables"); variables != "" {
		if err := jsonDecode(strings.NewReader(variables), &params.Variables); err != nil {
			return nil, fmt.Errorf("variables could not be decoded: %w", err)
		}
	}

	if extensions := form.Get("extensions"); extensions != "" {
		if err := jsonDecode(strings.NewReader(extensions), &params.Extensions); err != nil {
			return nil, fmt.Errorf("extensions could not be decoded: %w", err)
		}
	}

	return params, nil
}

func (h UrlEncodedForm) parseJson(bodyString string) (*graphql.RawParams, error) {
	params := &graphql.RawParams{}
	bodyReader := io.NopCloser(strings.NewReader(bodyString))

	err := jsonDecode(bodyReader, params)
	if err != nil {
		return nil, err
	}

	return params, nil
}
