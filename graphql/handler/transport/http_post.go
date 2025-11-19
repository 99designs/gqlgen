package transport

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"sync"

	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/99designs/gqlgen/graphql"
)

// POST implements the POST side of the default HTTP transport
// defined in https://github.com/APIs-guru/graphql-over-http#post
type POST struct {
	// Map of all headers that are added to graphql response. If not
	// set, only one header: Content-Type: application/graphql-response+json will be set.
	ResponseHeaders map[string][]string

	// UseGrapQLResponseJsonByDefault determines whether to use 'application/graphql-response+json'
	// as the response content type
	// when the Accept header is empty or 'application/*' or '*/*'.
	UseGrapQLResponseJsonByDefault bool
}

var _ graphql.Transport = POST{}

func (h POST) Supports(r *http.Request) bool {
	if r.Header.Get("Upgrade") != "" {
		return false
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return false
	}

	return r.Method == http.MethodPost && mediaType == "application/json"
}

func getRequestBody(r *http.Request) (string, error) {
	if r == nil || r.Body == nil {
		return "", nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", fmt.Errorf("unable to get Request Body %w", err)
	}
	return string(body), nil
}

var pool = sync.Pool{
	New: func() any {
		return &graphql.RawParams{}
	},
}

func (h POST) Do(w http.ResponseWriter, r *http.Request, exec graphql.GraphExecutor) {
	ctx := r.Context()
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
	params := pool.Get().(*graphql.RawParams)
	defer func() {
		params.Headers = nil
		params.ReadTime = graphql.TraceTiming{}
		params.Extensions = nil
		params.OperationName = ""
		params.Query = ""
		params.Variables = nil

		pool.Put(params)
	}()
	params.Headers = r.Header

	start := graphql.Now()
	params.ReadTime = graphql.TraceTiming{
		Start: start,
		End:   graphql.Now(),
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		gqlErr := gqlerror.Errorf("could not read request body: %+v", err)
		resp := exec.DispatchError(ctx, gqlerror.List{gqlErr})
		writeJson(w, resp)
		return
	}

	bodyReader := bytes.NewReader(bodyBytes)
	if err := jsonDecode(bodyReader, &params); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		gqlErr := gqlerror.Errorf(
			"json request body could not be decoded: %+v body:%s",
			err,
			string(bodyBytes),
		)
		resp := exec.DispatchError(ctx, gqlerror.List{gqlErr})
		writeJson(w, resp)
		return
	}

	rc, opErr := exec.CreateOperationContext(ctx, params)
	if opErr != nil {
		if contentType == acceptApplicationGraphqlResponseJson {
			w.WriteHeader(statusForGraphQLResponse(opErr))
		} else {
			w.WriteHeader(statusFor(opErr))
		}
		resp := exec.DispatchError(graphql.WithOperationContext(ctx, rc), opErr)
		writeJson(w, resp)
		return
	}

	var responses graphql.ResponseHandler
	responses, ctx = exec.DispatchOperation(ctx, rc)
	writeJson(w, responses(ctx))
}
