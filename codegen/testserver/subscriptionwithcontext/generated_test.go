//go:generate rm -f resolver.go
//go:generate go run ../../../testdata/gqlgen.go -config gqlgen.yml -stub stub.go

package subscriptionwithcontext

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

type ctxKey string

// observedContexts records the ctx parameter that AroundResponses sees for
// every subscription payload. Each observation captures the value of a known
// context key so tests can correlate it back to what the resolver attached.
type observedContexts struct {
	mu  sync.Mutex
	ids []string
}

func (o *observedContexts) record(ctx context.Context) {
	o.mu.Lock()
	defer o.mu.Unlock()
	id, _ := ctx.Value(ctxKey("event-id")).(string)
	o.ids = append(o.ids, id)
}

func (o *observedContexts) snapshot() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]string, len(o.ids))
	copy(out, o.ids)
	return out
}

func newTestServer(t *testing.T, resolvers *Stub, obs *observedContexts) *client.Client {
	t.Helper()
	srv := handler.New(NewExecutableSchema(Config{Resolvers: resolvers}))
	srv.AddTransport(transport.SSE{})
	srv.AddTransport(transport.POST{})
	srv.AroundResponses(func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
		obs.record(ctx)
		return next(ctx)
	})
	return client.New(srv)
}

func TestSubscriptionContext_MarkedFieldYieldsPerEventCtx(t *testing.T) {
	resolvers := &Stub{}
	resolvers.SubscriptionResolver.Marked = func(ctx context.Context) (<-chan graphql.Event[string], error) {
		ch := make(chan graphql.Event[string], 3)
		ch <- graphql.Event[string]{
			Context: context.WithValue(ctx, ctxKey("event-id"), "evt-A"),
			Value:   "a",
		}
		ch <- graphql.Event[string]{
			Context: context.WithValue(ctx, ctxKey("event-id"), "evt-B"),
			Value:   "b",
		}
		ch <- graphql.Event[string]{
			Context: context.WithValue(ctx, ctxKey("event-id"), "evt-C"),
			Value:   "c",
		}
		close(ch)
		return ch, nil
	}

	obs := &observedContexts{}
	c := newTestServer(t, resolvers, obs)

	read := c.SSE(context.Background(), `subscription { marked }`)
	defer read.Close()
	var got []string
	for len(got) < 3 {
		var resp struct {
			Data struct {
				Marked string
			}
			Label      string          `json:"label"`
			Path       []any           `json:"path"`
			HasNext    bool            `json:"hasNext"`
			Errors     json.RawMessage `json:"errors"`
			Extensions map[string]any  `json:"extensions"`
		}
		if err := read.Next(&resp); err != nil {
			t.Fatalf("SSE Next failed after %d payloads: %v", len(got), err)
		}
		got = append(got, resp.Data.Marked)
	}

	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("expected payloads [a b c], got %v", got)
	}
	ids := obs.snapshot()
	if len(ids) < 3 {
		t.Fatalf("expected at least 3 interceptor observations, got %d (%v)", len(ids), ids)
	}
	wantPrefix := []string{"evt-A", "evt-B", "evt-C"}
	for i, want := range wantPrefix {
		if ids[i] != want {
			t.Errorf("interceptor observation %d: got %q, want %q", i, ids[i], want)
		}
	}
}

func TestSubscriptionContext_UnmarkedFieldKeepsSubscriptionCtx(t *testing.T) {
	resolvers := &Stub{}
	resolvers.SubscriptionResolver.Unmarked = func(ctx context.Context) (<-chan string, error) {
		ch := make(chan string, 2)
		ch <- "x"
		ch <- "y"
		close(ch)
		return ch, nil
	}

	obs := &observedContexts{}
	c := newTestServer(t, resolvers, obs)

	read := c.SSE(context.Background(), `subscription { unmarked }`)
	defer read.Close()
	var got []string
	for len(got) < 2 {
		var resp struct {
			Data struct {
				Unmarked string
			}
			Label      string          `json:"label"`
			Path       []any           `json:"path"`
			HasNext    bool            `json:"hasNext"`
			Errors     json.RawMessage `json:"errors"`
			Extensions map[string]any  `json:"extensions"`
		}
		if err := read.Next(&resp); err != nil {
			t.Fatalf("SSE Next failed after %d payloads: %v", len(got), err)
		}
		got = append(got, resp.Data.Unmarked)
	}

	if len(got) != 2 || got[0] != "x" || got[1] != "y" {
		t.Fatalf("expected payloads [x y], got %v", got)
	}

	// Unmarked field: interceptor's ctx never carried "event-id" so all
	// observations should be the empty string.
	for i, id := range obs.snapshot() {
		if id != "" {
			t.Errorf("unmarked subscription leaked per-event id at obs %d: %q", i, id)
		}
	}
}
