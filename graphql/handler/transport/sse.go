package transport

import (
	"context"
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

type (
	SSE struct {
		KeepAlivePingInterval time.Duration
		// MinEventInterval optionally paces SSE event frames for high-throughput streams.
		MinEventInterval time.Duration
	}

	sseConnection struct {
		ctx             context.Context
		mu              sync.Mutex
		f               http.Flusher
		keepAliveTicker *time.Ticker
	}
)

var _ graphql.Transport = SSE{}

func (t SSE) Supports(r *http.Request) bool {
	if !strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return false
	}
	return r.Method == http.MethodPost && mediaType == "application/json"
}

func (t SSE) Do(w http.ResponseWriter, r *http.Request, exec graphql.GraphExecutor) {
	ctx := r.Context()
	flusher, ok := w.(http.Flusher)
	if !ok {
		SendErrorf(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	c := &sseConnection{
		ctx: ctx,
		f:   flusher,
	}

	defer c.flush()

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "application/json")

	params := &graphql.RawParams{}
	start := graphql.Now()

	bodyString, err := getRequestBody(r)
	if err != nil {
		gqlErr := gqlerror.Errorf("could not get json request body: %+v", err)
		resp := exec.DispatchError(ctx, gqlerror.List{gqlErr})
		log.Printf("could not get json request body: %+v", err.Error())
		writeJson(w, resp)
		return
	}

	bodyReader := io.NopCloser(strings.NewReader(bodyString))
	if err = jsonDecode(bodyReader, params); err != nil {
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

	params.Headers = r.Header
	params.ReadTime = graphql.TraceTiming{
		Start: start,
		End:   graphql.Now(),
	}

	rc, opErr := exec.CreateOperationContext(ctx, params)
	ctx = graphql.WithOperationContext(ctx, rc)
	c.ctx = ctx

	w.Header().Set("Content-Type", "text/event-stream")
	c.writeAndFlush(w, func(w io.Writer) {
		fmt.Fprint(w, ":\n\n")
	})

	lastEventSent := time.Time{}
	eventCtx := ctx
	writeEvent := func(ctx context.Context, write func(io.Writer)) bool {
		if !waitForMinEventInterval(ctx, lastEventSent, t.MinEventInterval) {
			return false
		}

		c.writeAndFlush(w, write)
		lastEventSent = time.Now()
		return true
	}

	if t.KeepAlivePingInterval > 0 {
		c.mu.Lock()
		c.keepAliveTicker = time.NewTicker(t.KeepAlivePingInterval)
		c.mu.Unlock()

		go c.keepAlive(w)
	}

	if opErr != nil {
		resp := exec.DispatchError(ctx, opErr)
		if !writeEvent(ctx, func(w io.Writer) {
			writeJsonWithSSE(w, resp)
		}) {
			return
		}
	} else {
		responses, dispatchCtx := exec.DispatchOperation(ctx, rc)
		eventCtx = dispatchCtx
		for {
			response := responses(dispatchCtx)
			if response == nil {
				break
			}
			if !writeEvent(dispatchCtx, func(w io.Writer) {
				writeJsonWithSSE(w, response)
			}) {
				return
			}

			c.resetTicker(t.KeepAlivePingInterval)
		}
	}

	writeEvent(eventCtx, func(w io.Writer) {
		fmt.Fprint(w, "event: complete\n\n")
	})
}

func waitForMinEventInterval(
	ctx context.Context,
	lastEventSent time.Time,
	minInterval time.Duration,
) bool {
	if minInterval <= 0 || lastEventSent.IsZero() {
		return true
	}

	wait := time.Until(lastEventSent.Add(minInterval))
	if wait <= 0 {
		return true
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (c *sseConnection) resetTicker(interval time.Duration) {
	if interval != 0 {
		c.mu.Lock()
		c.keepAliveTicker.Reset(interval)
		c.mu.Unlock()
	}
}

func (c *sseConnection) keepAlive(w io.Writer) {
	for {
		select {
		case <-c.ctx.Done():
			c.keepAliveTicker.Stop()
			return
		case <-c.keepAliveTicker.C:
			c.writeAndFlush(w, func(w io.Writer) {
				fmt.Fprint(w, ": ping\n\n")
			})
		}
	}
}

func (c *sseConnection) flush() {
	c.mu.Lock()
	c.f.Flush()
	c.mu.Unlock()
}

func (c *sseConnection) writeAndFlush(w io.Writer, write func(io.Writer)) {
	c.mu.Lock()
	write(w)
	c.f.Flush()
	c.mu.Unlock()
}

func writeJsonWithSSE(w io.Writer, response *graphql.Response) {
	b, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(w, "event: next\ndata: %s\n\n", b)
}
