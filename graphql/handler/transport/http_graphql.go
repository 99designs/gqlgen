package transport

import (
	"mime"
	"net/http"
	"net/url"
	"strings"

	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/99designs/gqlgen/graphql"
)

// GRAPHQL implements the application/graphql side of the HTTP transport
// see: https://graphql.org/learn/serving-over-http/#post-request
// If the "application/graphql" Content-Type header is present, treat
// the HTTP POST body contents as the GraphQL query string.
type GRAPHQL struct {
	// Map of all headers that are added to graphql response. If not
	// set, only one header: Content-Type: application/json will be set.
	ResponseHeaders map[string][]string
}

var _ graphql.Transport = GRAPHQL{}

func (h GRAPHQL) Supports(r *http.Request) bool {
	if r.Header.Get("Upgrade") != "" {
		return false
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return false
	}

	return r.Method == "POST" && mediaType == "application/graphql"
}

func (h GRAPHQL) Do(w http.ResponseWriter, r *http.Request, exec graphql.GraphExecutor) {
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
		gqlErr := gqlerror.Errorf("could not get request body: %+v", err)
		resp := exec.DispatchError(ctx, gqlerror.List{gqlErr})
		writeJson(w, resp)
		return
	}

	params.Query, err = cleanupBody(bodyString)
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

// Makes sure we strip "query=" keyword from body and
// that body is not url escaped
func cleanupBody(body string) (out string, err error) {
	// Some clients send 'query=' at the start of body payload. Let's remove
	// it to get GQL query only.
	body = strings.TrimPrefix(body, "query=")

	// Body payload can be url encoded or not. We check if %7B - "{" character
	// is where query starts. If it is, query is url encoded.
	if strings.HasPrefix(body, "%7B") {
		body, err = url.QueryUnescape(body)

		if err != nil {
			return body, err
		}
	}

	return body, err
}
