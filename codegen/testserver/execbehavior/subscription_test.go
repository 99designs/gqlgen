package execbehavior

import (
	"context"
	"errors"
	"io"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/codegen/testserver/execbehavior/internal/exectest"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
)

type eventKey struct{}

func newSubscriptionSchema() graphql.ExecutableSchema {
	resolvers := &Stub{}
	r := &resolvers.SubscriptionResolver
	r.Count = func(ctx context.Context, to int) (<-chan int, error) {
		ch := make(chan int, to)
		for i := 1; i <= to; i++ {
			ch <- i
		}
		close(ch)
		return ch, nil
	}
	r.Events = func(ctx context.Context, to int) (<-chan graphql.Event[int], error) {
		ch := make(chan graphql.Event[int], to)
		for i := 1; i <= to; i++ {
			// Odd events carry no context, and are paired with the context of the call.
			var eventCtx context.Context
			if i%2 == 0 {
				eventCtx = context.WithValue(ctx, eventKey{}, i)
			}
			ch <- graphql.Event[int]{Context: eventCtx, Value: i}
		}
		close(ch)
		return ch, nil
	}
	r.Failing = func(ctx context.Context) (<-chan int, error) {
		return nil, errors.New("failing stream")
	}
	r.FailingEvents = func(ctx context.Context) (<-chan graphql.Event[int], error) {
		return nil, errors.New("failing events")
	}
	r.NilStream = func(ctx context.Context) (<-chan int, error) { return make(chan int), nil }
	r.NullableNilStream = func(ctx context.Context) (<-chan *int, error) {
		return make(chan *int), nil
	}
	r.NilEvents = func(ctx context.Context) (<-chan graphql.Event[int], error) {
		return make(chan graphql.Event[int]), nil
	}
	r.Panicking = func(ctx context.Context) (<-chan int, error) { panic("panicking stream") }
	r.ReplacedStream = func(ctx context.Context) (<-chan *string, error) {
		return make(chan *string), nil
	}
	r.ReplacedEvents = func(ctx context.Context) (<-chan graphql.Event[*string], error) {
		return make(chan graphql.Event[*string]), nil
	}
	r.WrongType = func(ctx context.Context) (<-chan *int, error) { return make(chan *int), nil }
	r.WrongTypeEvents = func(ctx context.Context) (<-chan graphql.Event[*int], error) {
		return make(chan graphql.Event[*int]), nil
	}
	nilAnimal := func(ctx context.Context) (<-chan Animal, error) {
		ch := make(chan Animal, 1)
		ch <- nil
		close(ch)
		return ch, nil
	}
	r.NilAnimal = nilAnimal
	r.NullableNilAnimal = nilAnimal

	once := func(field string) func() graphql.Marshaler {
		sent := false
		return func() graphql.Marshaler {
			if sent {
				return nil
			}
			sent = true
			return graphql.WriterFunc(func(w io.Writer) {
				_, _ = io.WriteString(w, `{"`+field+`":"from directive"}`)
			})
		}
	}
	return NewExecutableSchema(Config{
		Resolvers: resolvers,
		Directives: DirectiveRoot{
			ReplaceStream: func(ctx context.Context, obj any, next graphql.Resolver, with string) (any, error) {
				if _, err := next(ctx); err != nil {
					return nil, err
				}
				switch with {
				case "stream":
					next := once("replacedStream")
					return func(context.Context) graphql.Marshaler { return next() }, nil
				case "events":
					next := once("replacedEvents")
					return func(ctx context.Context) (context.Context, graphql.Marshaler) {
						return ctx, next()
					}, nil
				case "nil":
					return nil, nil
				}
				return 42, nil
			},
		},
	})
}

func TestSubscription(t *testing.T) {
	es := newSubscriptionSchema()
	for _, tc := range []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "one response per event",
			query: `subscription { count(to: 3) }`,
			want:  `[{"data":{"count":1}},{"data":{"count":2}},{"data":{"count":3}}]`,
		},
		{
			name:  "events with and without their own context",
			query: `subscription { events(to: 3) }`,
			want:  `[{"data":{"events":1}},{"data":{"events":2}},{"data":{"events":3}}]`,
		},
		{
			name:  "alias",
			query: `subscription { n: count(to: 1) }`,
			want:  `[{"data":{"n":1}}]`,
		},
		{
			name:  "resolver error",
			query: `subscription { failing }`,
			want:  `[{"errors":[{"message":"failing stream","path":["failing"]}],"data":null}]`,
		},
		{
			name:  "resolver error of a field with event context",
			query: `subscription { failingEvents }`,
			want:  `[{"errors":[{"message":"failing events","path":["failingEvents"]}],"data":null}]`,
		},
		{
			name:  "nil for a non-null field",
			query: `subscription { nilStream }`,
			want: `[{"errors":[{"message":"cannot return null for non-null field ` +
				`Subscription.nilStream (Int!): the resolver returned nil",` +
				`"path":["nilStream"]}],"data":null}]`,
		},
		{
			// KNOWN BUG: no error is reported, and the handler then calls the nil stream,
			// which panics.
			// Expected: an error, as for the non-null field, because a nil stream never
			// delivers an event, so the subscription cannot start.
			// Issue: https://github.com/99designs/gqlgen/issues/4455
			name:  "nil for a nullable field",
			query: `subscription { nullableNilStream }`,
			want:  `"panic: runtime error: invalid memory address or nil pointer dereference"`,
		},
		{
			name:  "nil for a non-null field with event context",
			query: `subscription { nilEvents }`,
			want: `[{"errors":[{"message":"cannot return null for non-null field ` +
				`Subscription.nilEvents (Int!): the resolver returned nil",` +
				`"path":["nilEvents"]}],"data":null}]`,
		},
		{
			// KNOWN BUG: the event is sent as null, with an error that has no path and does
			// not name the field.
			// Expected: data is null and the error has the path of the field, because the
			// spec propagates a null in a non-null field to the nearest nullable parent,
			// and asks an error of a field to give its path.
			// Issue: https://github.com/99designs/gqlgen/issues/4464
			name:  "nil event for a non-null interface",
			query: `subscription { nilAnimal { name } }`,
			want: `[{"errors":[{"message":"cannot return null for non-null position (Animal!): ` +
				`the value was nil"}],"data":{"nilAnimal":null}}]`,
		},
		{
			name:  "nil event for a nullable interface",
			query: `subscription { nullableNilAnimal { name } }`,
			want:  `[{"data":{"nullableNilAnimal":null}}]`,
		},
		{
			name:  "resolver panic",
			query: `subscription { panicking }`,
			want:  `[{"errors":[{"message":"internal system error","path":["panicking"]}],"data":null}]`,
		},
		{
			name:  "the only field is skipped",
			query: `subscription { count(to: 1) @skip(if: true) }`,
			want:  `[{"errors":[{"message":"must subscribe to exactly one stream"}],"data":null}]`,
		},
		{
			name:  "directive returns the stream",
			query: `subscription { replacedStream }`,
			want:  `[{"data":{"replacedStream":"from directive"}}]`,
		},
		{
			name:  "directive returns the stream with event context",
			query: `subscription { replacedEvents }`,
			want:  `[{"data":{"replacedEvents":"from directive"}}]`,
		},
		{
			name:  "directive returns the wrong type",
			query: `subscription { wrongType }`,
			want: `[{"errors":[{"message":"unexpected type int from middleware/directive chain, ` +
				`should be \u003c-chan *int","path":["wrongType"]}],"data":null}]`,
		},
		{
			name:  "directive returns the wrong type for a field with event context",
			query: `subscription { wrongTypeEvents }`,
			want: `[{"errors":[{"message":"unexpected type int from middleware/directive chain, ` +
				`should be \u003c-chan graphql.Event[*int]","path":["wrongTypeEvents"]}],"data":null}]`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.JSONEq(t, tc.want, subscribe(t, es, tc.query))
		})
	}
}

// subscribe runs the subscription query on es with exectest.Subscribe, with the panics of
// the resolvers recovered as "internal system error".
func subscribe(t *testing.T, es graphql.ExecutableSchema, query string) string {
	t.Helper()

	exec := executor.New(es)
	exec.SetRecoverFunc(func(ctx context.Context, err any) error {
		return errors.New("internal system error")
	})
	return exectest.Subscribe(t, exec, query)
}

func TestSubscriptionEndsWithContext(t *testing.T) {
	start := func(t *testing.T) (graphql.ResponseHandler, context.Context) {
		resolvers := &Stub{}
		silent := make(chan graphql.Event[*int])
		resolvers.SubscriptionResolver.SilentEvents = func(ctx context.Context) (<-chan graphql.Event[*int], error) {
			return silent, nil
		}
		exec := executor.New(NewExecutableSchema(Config{Resolvers: resolvers}))
		ctx := graphql.StartOperationTrace(context.Background())
		opCtx, errs := exec.CreateOperationContext(ctx, &graphql.RawParams{
			Query: `subscription { silentEvents }`,
		})
		require.Empty(t, errs)
		return exec.DispatchOperation(ctx, opCtx)
	}

	t.Run("cancelled before the next event is asked for", func(t *testing.T) {
		handler, ctx := start(t)
		ctx, cancel := context.WithCancel(ctx)
		cancel()
		require.Nil(t, handler(ctx))
	})

	t.Run("cancelled while waiting for the next event", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			handler, ctx := start(t)
			ctx, cancel := context.WithCancel(ctx)
			go func() {
				// Cancel once the handler waits for an event that never comes.
				synctest.Wait()
				cancel()
			}()
			require.Nil(t, handler(ctx))
		})
	})
}
