package transport

import (
	"context"
	"mime"
	"net/http"

	"github.com/99designs/gqlgen/graphql"
)

// POST implements the POST side of the default HTTP transport
// defined in https://github.com/APIs-guru/graphql-over-http#post
type POST struct {
	StatusCodeFunc func(ctx context.Context, resp *graphql.Response) int
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

	return r.Method == "POST" && mediaType == "application/json"
}

func (h POST) Do(w http.ResponseWriter, r *http.Request, exec graphql.GraphExecutor) {
	w.Header().Set("Content-Type", "application/json")

	var params *graphql.RawParams
	if err := jsonDecode(r.Body, &params); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJsonErrorf(w, "json body could not be decoded: "+err.Error())
		return
	}

	rc, err := exec.CreateOperationContext(r.Context(), params)
	if err != nil {
		ctx := graphql.WithOperationContext(r.Context(), rc)
		resp := exec.DispatchError(ctx, err)
		h.writeStatusCode(ctx, w, resp)
		writeJson(w, resp)
		return
	}
	responses, ctx := exec.DispatchOperation(r.Context(), rc)
	resp := responses(ctx)
	h.writeStatusCode(ctx, w, resp)
	writeJson(w, resp)
}

func (h POST) writeStatusCode(ctx context.Context, w http.ResponseWriter, resp *graphql.Response) {
	if h.StatusCodeFunc == nil {
		w.WriteHeader(httpStatusCode(resp))
	} else {
		w.WriteHeader(h.StatusCodeFunc(ctx, resp))
	}
}
