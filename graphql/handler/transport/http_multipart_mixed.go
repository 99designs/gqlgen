package transport

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"strings"

	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/99designs/gqlgen/graphql"
)

// MultipartMixed is a transport that supports the multipart/mixed spec
type MultipartMixed struct {
	Boundary string
}

var _ graphql.Transport = MultipartMixed{}

// Supports checks if the request supports the multipart/mixed spec
// Might be worth check the spec required, but Apollo Client mislabel the spec in the headers.
func (t MultipartMixed) Supports(r *http.Request) bool {
	if !strings.Contains(r.Header.Get("Accept"), "multipart/mixed") {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return false
	}
	return r.Method == http.MethodPost && mediaType == "application/json"
}

// Do implements the multipart/mixed spec as a multipart/mixed response
func (t MultipartMixed) Do(w http.ResponseWriter, r *http.Request, exec graphql.GraphExecutor) {
	// Implements the multipart/mixed spec as a multipart/mixed response:
	// * https://github.com/graphql/graphql-wg/blob/e4ef5f9d5997815d9de6681655c152b6b7838b4c/rfcs/DeferStream.md
	//   2022/08/23 as implemented by gqlgen.
	// * https://github.com/graphql/graphql-wg/blob/f22ea7748c6ebdf88fdbf770a8d9e41984ebd429/rfcs/DeferStream.md June 2023 Spec for the
	//   `incremental` field
	// Follows the format that is used in the Apollo Client tests:
	// https://github.com/apollographql/apollo-client/blob/v3.11.8/src/link/http/__tests__/responseIterator.ts#L68
	// Apollo Client, despite mentioning in its requests that they require the 2022 spec, it wants the
	// `incremental` field to be an array of responses, not a single response. Theoretically we could
	// batch responses in the `incremental` field, if we wanted to optimize this code.
	ctx := r.Context()
	flusher, ok := w.(http.Flusher)
	if !ok {
		SendErrorf(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	defer flusher.Flush()

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// This header will be replaced below, but it's required in case we return errors.
	w.Header().Set("Content-Type", "application/json")

	boundary := t.Boundary
	if boundary == "" {
		boundary = "graphql"
	}

	params := &graphql.RawParams{}
	start := graphql.Now()
	params.Headers = r.Header
	params.ReadTime = graphql.TraceTiming{
		Start: start,
		End:   graphql.Now(),
	}

	bodyString, err := getRequestBody(r)
	if err != nil {
		gqlErr := gqlerror.Errorf("could not get json request body: %+v", err)
		resp := exec.DispatchError(ctx, gqlerror.List{gqlErr})
		log.Printf("could not get json request body: %+v", err.Error())
		writeJson(w, resp)
		return
	}

	bodyReader := io.NopCloser(strings.NewReader(bodyString))
	if err = jsonDecode(bodyReader, &params); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		gqlErr := gqlerror.Errorf(
			"json request body could not be decoded: %+v body:%s",
			err,
			bodyString,
		)
		resp := exec.DispatchError(ctx, gqlerror.List{gqlErr})
		log.Printf("decoding error: %+v body:%s", err.Error(), bodyString)
		writeJson(w, resp)
		return
	}

	rc, opErr := exec.CreateOperationContext(ctx, params)
	ctx = graphql.WithOperationContext(ctx, rc)

	// Example of the response format (note the new lines are important!):
	// --graphql
	// Content-Type: application/json
	//
	// {"data":{"apps":{"apps":[ .. ],"totalNumApps":161,"__typename":"AppsOutput"}},"hasNext":true}
	//
	// --graphql
	// Content-Type: application/json
	//
	// {"incremental":[{"data":{"groupAccessCount":0},"label":"test","path":["apps","apps",7],"hasNext":true}],"hasNext":true}

	if opErr != nil {
		w.WriteHeader(statusFor(opErr))

		resp := exec.DispatchError(ctx, opErr)
		writeJson(w, resp)
		return
	}

	w.Header().Set(
		"Content-Type",
		fmt.Sprintf(`multipart/mixed;boundary="%s";deferSpec=20220824`, boundary),
	)

	responses, ctx := exec.DispatchOperation(ctx, rc)
	initialResponse := true
	for {
		response := responses(ctx)
		if response == nil {
			break
		}

		fmt.Fprintf(w, "--%s\r\n", boundary)
		fmt.Fprintf(w, "Content-Type: application/json\r\n\r\n")

		if initialResponse {
			writeJson(w, response)
			initialResponse = false
		} else {
			writeIncrementalJson(w, response, response.HasNext)
		}
		fmt.Fprintf(w, "\r\n\r\n")
		flusher.Flush()
	}
}

func writeIncrementalJson(w io.Writer, response *graphql.Response, hasNext *bool) {
	// TODO: Remove this wrapper on response once gqlgen supports the 2023 spec
	b, err := json.Marshal(struct {
		Incremental []graphql.Response `json:"incremental"`
		HasNext     *bool              `json:"hasNext"`
	}{
		Incremental: []graphql.Response{*response},
		HasNext:     hasNext,
	})
	if err != nil {
		panic(err)
	}
	w.Write(b)
}
