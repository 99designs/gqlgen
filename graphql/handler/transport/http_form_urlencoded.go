package transport

import (
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

	return r.Method == "POST" && mediaType == "application/x-www-form-urlencoded"
}

func (h UrlEncodedForm) Do(w http.ResponseWriter, r *http.Request, exec graphql.GraphExecutor) {
	ctx := r.Context()
	writeHeaders(w, h.ResponseHeaders)
	params := &graphql.RawParams{}
	start := graphql.Now()
	params.Headers = r.Header
	params.ReadTime = graphql.TraceTiming{
		Start: start,
		End:   graphql.Now(),
	}

	bodyString, err := getRequestBody(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		gqlErr := gqlerror.Errorf("could not get form body: %+v", err)
		resp := exec.DispatchError(ctx, gqlerror.List{gqlErr})
		writeJson(w, resp)
		return
	}

	params, err = h.parseBody(bodyString)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		gqlErr := gqlerror.Errorf("could not cleanup body: %+v", err)
		resp := exec.DispatchError(ctx, gqlerror.List{gqlErr})
		writeJson(w, resp)
		return
	}

	rc, OpErr := exec.CreateOperationContext(ctx, params)
	if OpErr != nil {
		w.WriteHeader(statusFor(OpErr))
		resp := exec.DispatchError(graphql.WithOperationContext(ctx, rc), OpErr)
		writeJson(w, resp)
		return
	}

	var responses graphql.ResponseHandler
	responses, ctx = exec.DispatchOperation(ctx, rc)
	writeJson(w, responses(ctx))
}

func (h UrlEncodedForm) parseBody(bodyString string) (*graphql.RawParams, error) {
	switch {
	case strings.Contains(bodyString, "\"query\":"):
		// body is json
		return h.parseJson(bodyString)
	case strings.HasPrefix(bodyString, "query=%7B"):
		// body is urlencoded
		return h.parseEncoded(bodyString)
	default:
		// body is plain text
		params := &graphql.RawParams{}
		params.Query = strings.TrimPrefix(bodyString, "query=")

		return params, nil
	}
}

func (h UrlEncodedForm) parseEncoded(bodyString string) (*graphql.RawParams, error) {
	params := &graphql.RawParams{}

	query, err := url.QueryUnescape(bodyString)
	if err != nil {
		return nil, err
	}

	params.Query = strings.TrimPrefix(query, "query=")

	return params, nil
}

func (h UrlEncodedForm) parseJson(bodyString string) (*graphql.RawParams, error) {
	params := &graphql.RawParams{}
	bodyReader := io.NopCloser(strings.NewReader(bodyString))

	err := jsonDecode(bodyReader, &params)
	if err != nil {
		return nil, err
	}

	return params, nil
}
