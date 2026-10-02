package executor_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
)

// dispatchWithEventContext is the subscription path taken when a generated
// schema implements the optional ExecutableSchemaWithEventContext interface. It
// exists so that AroundResponses interceptors receive the per-event context
// rather than the context the subscription was opened with, which is what lets
// a tracing or logging extension attribute work to the event that caused it.
//
// Nothing in the repository mocks that optional interface — only generated
// testservers implement it — so these tests define the fake they need, with
// Func fields and Invoked booleans so a test can assert which dispatch path
// ran. That is the contract: a subscription takes the event path and a query
// does not, even on a schema that offers both.

// eventCtxKey is the type of the value the fake schema plants on each event
// context, so a test can prove which context reached the response middleware.
type eventCtxKey struct{}

// fakeEventSchema implements both graphql.ExecutableSchema and the optional
// graphql.ExecutableSchemaWithEventContext.
type fakeEventSchema struct {
	schema *ast.Schema

	// ExecFunc and ExecWithEventContextFunc stand in for the two dispatch
	// entry points. Either may be nil, in which case calling it is a test
	// failure rather than a panic in production code.
	ExecFunc                 func(ctx context.Context) graphql.ResponseHandler
	ExecWithEventContextFunc func(ctx context.Context) graphql.ResponseHandlerWithContext

	ExecInvoked                 bool
	ExecWithEventContextInvoked bool
}

func (s *fakeEventSchema) Schema() *ast.Schema { return s.schema }

func (s *fakeEventSchema) Complexity(
	context.Context, string, string, int, map[string]any,
) (int, bool) {
	return 0, false
}

func (s *fakeEventSchema) Exec(ctx context.Context) graphql.ResponseHandler {
	s.ExecInvoked = true
	if s.ExecFunc == nil {
		return graphql.OneShot(&graphql.Response{Data: []byte(`{"name":"default"}`)})
	}
	return s.ExecFunc(ctx)
}

func (s *fakeEventSchema) ExecWithEventContext(
	ctx context.Context,
) graphql.ResponseHandlerWithContext {
	s.ExecWithEventContextInvoked = true
	if s.ExecWithEventContextFunc == nil {
		return func(ctx context.Context) (context.Context, *graphql.Response) {
			return ctx, nil
		}
	}
	return s.ExecWithEventContextFunc(ctx)
}

// newEventSchema returns a fake over a schema with a query and a subscription,
// so one executor can be driven down either dispatch path.
func newEventSchema(tb testing.TB) *fakeEventSchema {
	tb.Helper()
	return &fakeEventSchema{
		schema: gqlparser.MustLoadSchema(&ast.Source{Name: "event", Input: `
			type Query { name: String! }
			type Subscription { name: String! }
		`}),
	}
}

// dispatch creates an operation context for q and runs it, returning the
// response handler so a test can pull events from it.
func dispatch(
	tb testing.TB,
	exec *executor.Executor,
	q string,
) (graphql.ResponseHandler, context.Context) {
	tb.Helper()

	ctx := graphql.StartOperationTrace(context.Background())
	opCtx, errs := exec.CreateOperationContext(ctx, &graphql.RawParams{Query: q})
	require.Empty(tb, errs)
	return exec.DispatchOperation(ctx, opCtx)
}

// The documented contract: a subscription on a schema implementing the optional
// interface takes the event path, and a query on the same schema does not.
func TestDispatchChoosesTheEventPathForSubscriptionsOnly(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		query         string
		wantEventPath bool
		wantPlainExec bool
	}{
		"a subscription uses the event-aware path": {
			query:         "subscription { name }",
			wantEventPath: true,
			wantPlainExec: false,
		},
		"a query uses the default path": {
			query:         "{ name }",
			wantEventPath: false,
			wantPlainExec: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			es := newEventSchema(t)
			handler, ctx := dispatch(t, executor.New(es), tc.query)
			handler(ctx)

			assert.Equal(t, tc.wantEventPath, es.ExecWithEventContextInvoked,
				"ExecWithEventContext invocation")
			assert.Equal(t, tc.wantPlainExec, es.ExecInvoked, "Exec invocation")
		})
	}
}

// The point of the whole path: the response middleware must run against the
// context the event carried, not the one the subscription was opened with.
// Without this, an extension cannot tell one event from another.
func TestEventContextReachesTheResponseMiddleware(t *testing.T) {
	t.Parallel()

	es := newEventSchema(t)
	es.ExecWithEventContextFunc = func(context.Context) graphql.ResponseHandlerWithContext {
		sent := false
		return func(ctx context.Context) (context.Context, *graphql.Response) {
			if sent {
				return ctx, nil
			}
			sent = true
			// The value below is what proves the middleware saw this context
			// rather than the outer one.
			eventCtx := context.WithValue(ctx, eventCtxKey{}, "from-event")
			return eventCtx, &graphql.Response{Data: []byte(`{"name":"tick"}`)}
		}
	}

	exec := executor.New(es)
	var seen any
	exec.Use(
		aroundResponses(func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
			seen = ctx.Value(eventCtxKey{})
			return next(ctx)
		}),
	)

	handler, ctx := dispatch(t, exec, "subscription { name }")
	resp := handler(ctx)

	require.NotNil(t, resp)
	assert.JSONEq(t, `{"name":"tick"}`, string(resp.Data))
	assert.Equal(t, "from-event", seen,
		"the response middleware should receive the per-event context")
}

// A nil response ends the stream. Returning a response built from it instead
// would hand the transport an empty message to send after the subscription had
// already finished.
func TestEventDispatchStopsOnANilResponse(t *testing.T) {
	t.Parallel()

	es := newEventSchema(t)
	es.ExecWithEventContextFunc = func(context.Context) graphql.ResponseHandlerWithContext {
		return func(ctx context.Context) (context.Context, *graphql.Response) {
			return ctx, nil
		}
	}

	handler, ctx := dispatch(t, executor.New(es), "subscription { name }")
	assert.Nil(t, handler(ctx), "a nil event response should end the stream")
}

// A cancelled context stops the stream before the schema is asked for another
// event, so a client that has gone away costs nothing further.
func TestEventDispatchStopsOnACancelledContext(t *testing.T) {
	t.Parallel()

	es := newEventSchema(t)
	events := 0
	es.ExecWithEventContextFunc = func(context.Context) graphql.ResponseHandlerWithContext {
		return func(ctx context.Context) (context.Context, *graphql.Response) {
			events++
			return ctx, &graphql.Response{Data: []byte(`{"name":"tick"}`)}
		}
	}

	handler, ctx := dispatch(t, executor.New(es), "subscription { name }")

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	assert.Nil(t, handler(cancelled), "a cancelled context should end the stream")
	assert.Zero(t, events, "no event should be pulled after cancellation")
}

// An error raised while opening the subscription is reported once and the
// stream does not start, because there is nothing to stream.
func TestEventDispatchReportsErrorsRaisedWhileOpening(t *testing.T) {
	t.Parallel()

	es := newEventSchema(t)
	es.ExecWithEventContextFunc = func(ctx context.Context) graphql.ResponseHandlerWithContext {
		graphql.AddError(ctx, gqlerror.Errorf("could not subscribe"))
		return func(ctx context.Context) (context.Context, *graphql.Response) {
			return ctx, &graphql.Response{Data: []byte(`{"name":"should not be sent"}`)}
		}
	}

	handler, ctx := dispatch(t, executor.New(es), "subscription { name }")

	resp := handler(ctx)
	require.NotNil(t, resp)
	require.Len(t, resp.Errors, 1)
	assert.Equal(t, "could not subscribe", resp.Errors[0].Message)
	assert.Empty(t, string(resp.Data), "a subscription that failed to open sends no data")

	// OneShot delivers exactly once, so a transport looping on the handler
	// stops rather than repeating the error forever.
	assert.Nil(t, handler(ctx), "the error response should be delivered once")
}

// Errors and extensions a resolver records during one event belong to that
// event's response rather than leaking into the next one.
func TestEventDispatchCollectsPerEventErrorsAndExtensions(t *testing.T) {
	t.Parallel()

	es := newEventSchema(t)
	event := 0
	es.ExecWithEventContextFunc = func(context.Context) graphql.ResponseHandlerWithContext {
		return func(ctx context.Context) (context.Context, *graphql.Response) {
			event++
			switch event {
			case 1:
				graphql.AddError(ctx, gqlerror.Errorf("first event failed"))
				graphql.RegisterExtension(ctx, "eventNumber", 1)
				return ctx, &graphql.Response{Data: []byte(`{"name":"one"}`)}
			case 2:
				graphql.RegisterExtension(ctx, "eventNumber", 2)
				return ctx, &graphql.Response{Data: []byte(`{"name":"two"}`)}
			default:
				return ctx, nil
			}
		}
	}

	handler, ctx := dispatch(t, executor.New(es), "subscription { name }")

	first := handler(ctx)
	require.NotNil(t, first)
	require.Len(t, first.Errors, 1)
	assert.Equal(t, "first event failed", first.Errors[0].Message)
	assert.Equal(t, 1, first.Extensions["eventNumber"])

	second := handler(ctx)
	require.NotNil(t, second)
	assert.Empty(t, second.Errors, "the first event's error should not repeat")
	assert.Equal(t, 2, second.Extensions["eventNumber"])

	assert.Nil(t, handler(ctx), "the stream ends after the second event")
}

// A schema that does not implement the optional interface keeps the default
// path for subscriptions too, which is what makes the interface optional.
func TestSubscriptionWithoutTheOptionalInterfaceUsesTheDefaultPath(t *testing.T) {
	t.Parallel()

	es := newEventSchema(t)
	plain := &plainSchema{fakeEventSchema: es}

	handler, ctx := dispatch(t, executor.New(plain), "subscription { name }")
	resp := handler(ctx)

	require.NotNil(t, resp)
	assert.True(t, es.ExecInvoked, "the default path should have been used")
	assert.False(t, es.ExecWithEventContextInvoked)
}

// plainSchema exposes only graphql.ExecutableSchema, hiding the fake's
// ExecWithEventContext so the executor's type assertion fails.
type plainSchema struct {
	*fakeEventSchema
}

func (s *plainSchema) ExecWithEventContext() {}

// aroundResponses adapts a function into the HandlerExtension the executor's
// Use method takes.
func aroundResponses(f graphql.ResponseMiddleware) graphql.HandlerExtension {
	return responseExtension{f}
}

type responseExtension struct {
	f graphql.ResponseMiddleware
}

func (responseExtension) ExtensionName() string { return "testAroundResponses" }

func (responseExtension) Validate(graphql.ExecutableSchema) error { return nil }

func (e responseExtension) InterceptResponse(
	ctx context.Context,
	next graphql.ResponseHandler,
) *graphql.Response {
	return e.f(ctx, next)
}
