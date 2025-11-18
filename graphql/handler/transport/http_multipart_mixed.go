package transport

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/99designs/gqlgen/graphql"
)

// MultipartMixed is a transport that supports the multipart/mixed spec
type MultipartMixed struct {
	Boundary        string
	DeliveryTimeout time.Duration
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
	// *
	// https://github.com/graphql/graphql-wg/blob/e4ef5f9d5997815d9de6681655c152b6b7838b4c/rfcs/DeferStream.md
	//   2022/08/23 as implemented by gqlgen.
	// *
	// https://github.com/graphql/graphql-wg/blob/f22ea7748c6ebdf88fdbf770a8d9e41984ebd429/rfcs/DeferStream.md
	// June 2023 Spec for the
	//   `incremental` field
	// * https://github.com/graphql/graphql-over-http/blob/main/rfcs/IncrementalDelivery.md
	//   multipart specification
	// Follows the format that is used in the Apollo Client tests:
	// https://github.com/apollographql/apollo-client/blob/v3.11.8/src/link/http/__tests__/responseIterator.ts#L68
	// Apollo Client, despite mentioning in its requests that they require the 2022 spec, it wants
	// the `incremental` field to be an array of responses, not a single response. Theoretically we
	// could
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
		boundary = "-"
	}
	timeout := t.DeliveryTimeout
	if timeout.Milliseconds() < 1 {
		// If the timeout is less than 1ms, we'll set it to 1ms to avoid a busy loop
		timeout = 1 * time.Millisecond
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
	if opErr != nil {
		w.WriteHeader(statusFor(opErr))

		resp := exec.DispatchError(ctx, opErr)
		writeJson(w, resp)
		return
	}

	// Example of the response format (note the new lines and boundaries are important!):
	// https://github.com/graphql/graphql-over-http/blob/main/rfcs/IncrementalDelivery.md
	// --graphql
	// Content-Type: application/json
	//
	// {"data":{"apps":{"apps":[ .. ],"totalNumApps":161,"__typename":"AppsOutput"}},"hasNext":true}
	// --graphql
	// Content-Type: application/json
	//
	// {"incremental":[{"data":{"groupAccessCount":0},"label":"test","path":["apps","apps",7],"hasNext":true}],"hasNext":true}
	// --graphql
	// ...
	// --graphql--
	// Last boundary is a closing boundary with two dashes at the end.

	w.Header().Set(
		"Content-Type",
		fmt.Sprintf(`multipart/mixed;boundary="%s";deferSpec=20220824`, boundary),
	)

	a := newMultipartResponseAggregator(w, boundary, timeout)
	defer a.Done(w)

	responses, ctx := exec.DispatchOperation(ctx, rc)
	initialResponse := true
	for {
		response := responses(ctx)
		if response == nil {
			break
		}

		a.Add(response, initialResponse)
		initialResponse = false
	}
}

func writeIncrementalJson(w io.Writer, responses []*graphql.Response, hasNext bool) {
	// TODO: Remove this wrapper on response once gqlgen supports the 2023 spec
	b, err := json.Marshal(struct {
		Incremental []*graphql.Response `json:"incremental"`
		HasNext     bool                `json:"hasNext"`
	}{
		Incremental: responses,
		HasNext:     hasNext,
	})
	if err != nil {
		panic(err)
	}
	w.Write(b)
}

func writeBoundary(w io.Writer, boundary string, finalResponse bool) {
	if finalResponse {
		fmt.Fprintf(w, "--%s--\r\n", boundary)
		return
	}
	fmt.Fprintf(w, "--%s\r\n", boundary)
}

func writeContentTypeHeader(w io.Writer) {
	fmt.Fprintf(w, "Content-Type: application/json\r\n\r\n")
}

// multipartResponseAggregator helps us reduce the number of responses sent to the frontend by
// batching all the
// incremental responses together.
type multipartResponseAggregator struct {
	mu              sync.Mutex
	boundary        string
	initialResponse *graphql.Response
	deferResponses  []*graphql.Response
	done            chan bool
}

// newMultipartResponseAggregator creates a new multipartResponseAggregator
// The aggregator will flush responses to the client every `tickerDuration` (default 1ms) so that
// multiple incremental responses are batched together.
func newMultipartResponseAggregator(
	w http.ResponseWriter,
	boundary string,
	tickerDuration time.Duration,
) *multipartResponseAggregator {
	a := &multipartResponseAggregator{
		boundary: boundary,
		done:     make(chan bool, 1),
	}
	go func() {
		ticker := time.NewTicker(tickerDuration)
		defer ticker.Stop()
		for {
			select {
			case <-a.done:
				return
			case <-ticker.C:
				a.flush(w)
			}
		}
	}()
	return a
}

// Done flushes the remaining responses
func (a *multipartResponseAggregator) Done(w http.ResponseWriter) {
	a.done <- true
	a.flush(w)
}

// Add accumulates the responses
func (a *multipartResponseAggregator) Add(resp *graphql.Response, initialResponse bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if initialResponse {
		a.initialResponse = resp
		return
	}
	a.deferResponses = append(a.deferResponses, resp)
}

// flush sends the accumulated responses to the client
func (a *multipartResponseAggregator) flush(w http.ResponseWriter) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// If we don't have any responses, we can return early
	if a.initialResponse == nil && len(a.deferResponses) == 0 {
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		// This should never happen, as we check for this much earlier on
		panic("response writer does not support flushing")
	}

	hasNext := false
	if a.initialResponse != nil {
		// Initial response will need to begin with the boundary
		writeBoundary(w, a.boundary, false)
		writeContentTypeHeader(w)

		writeJson(w, a.initialResponse)
		hasNext = a.initialResponse.HasNext != nil && *a.initialResponse.HasNext

		// Handle when initial is aggregated with deferred responses.
		if len(a.deferResponses) > 0 {
			fmt.Fprintf(w, "\r\n")
			writeBoundary(w, a.boundary, false)
		}

		// Reset the initial response so we don't send it again
		a.initialResponse = nil
	}

	if len(a.deferResponses) > 0 {
		writeContentTypeHeader(w)

		// Note: while the 2023 spec that includes "incremental" does not
		// explicitly list the fields that should be included as part of the
		// incremental object, it shows hasNext only on the response payload
		// (marking the status of the operation as a whole), and instead the
		// response payload implements pending and complete fields to mark the
		// status of the incrementally delivered data.
		//
		// TODO: use the "HasNext" status of deferResponses items to determine
		// the operation status and pending / complete fields, but remove from
		// the incremental (deferResponses) object.
		hasNext = a.deferResponses[len(a.deferResponses)-1].HasNext != nil &&
			*a.deferResponses[len(a.deferResponses)-1].HasNext
		writeIncrementalJson(w, a.deferResponses, hasNext)

		// Reset the deferResponses so we don't send them again
		a.deferResponses = nil
	}

	// Make sure to put the delimiter after every request, so that Apollo Client knows that the
	// current payload has been sent, and updates the UI. This is particular important for the first
	// response and the last response, which may either hang or never get handled.
	// Final response will have a closing boundary with two dashes at the end.
	fmt.Fprintf(w, "\r\n")
	writeBoundary(w, a.boundary, !hasNext)
	flusher.Flush()
}
