//go:generate rm -f resolver.go
//go:generate go run ../../../testdata/gqlgen.go -config gqlgen.yml -stub stub.go

package directiveargnames

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

// The generated code unmarshals the arguments of a directive into variables named after the
// arguments, beside the object it passes to the directive and the values it writes for the
// other arguments. An argument with the name of one of those hides it.
func TestDirectiveArgumentNames(t *testing.T) {
	var (
		mu    sync.Mutex
		calls []string
	)
	resolvers := &Stub{}
	resolvers.QueryResolver.Root = func(ctx context.Context) (*string, error) {
		return nil, nil
	}
	resolvers.QueryResolver.Probed = func(
		ctx context.Context, x *string, input *ProbeInput,
	) (*string, error) {
		return x, nil
	}
	resolvers.QueryResolver.Literals = func(ctx context.Context, x *string) (*string, error) {
		return x, nil
	}
	srv := handler.New(NewExecutableSchema(Config{
		Resolvers: resolvers,
		Directives: DirectiveRoot{
			Probe: func(
				ctx context.Context, obj any, next graphql.Resolver,
				rawArgs, asMap, it *string, trueArg, falseArg *bool, fooBar *string, on *bool,
				list []*string,
			) (any, error) {
				got := []string{"obj=" + show(obj)}
				for _, a := range []struct {
					name  string
					value string
				}{
					{"rawArgs", deref(rawArgs)}, {"asMap", deref(asMap)}, {"it", deref(it)},
					{"true", deref(trueArg)}, {"false", deref(falseArg)},
					{"foo_bar", deref(fooBar)}, {"on", deref(on)},
				} {
					if a.value != "nil" {
						got = append(got, a.name+"="+a.value)
					}
				}
				if list != nil {
					values := make([]string, len(list))
					for i, v := range list {
						values[i] = deref(v)
					}
					got = append(got, fmt.Sprintf("list=%v", values))
				}
				mu.Lock()
				defer mu.Unlock()
				calls = append(calls, strings.Join(got, " "))
				return next(ctx)
			},
		},
	}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	t.Run("root field", func(t *testing.T) {
		calls = nil
		var resp struct{ Root *string }
		require.NoError(t, c.Post(`{ root }`, &resp))
		require.Equal(t, []string{
			"obj=<nil> list=[l]",
		}, calls)
	})

	t.Run("argument", func(t *testing.T) {
		calls = nil
		var resp struct{ Probed *string }
		require.NoError(t, c.Post(`{ probed(x: "a") }`, &resp))
		// The variable rawArgs hides the arguments of the field, so the directive receives
		// the value of its argument rawArgs as the object.
		require.Equal(t, []string{
			"obj=&r rawArgs=r foo_bar=fb",
		}, calls)
	})

	t.Run("boolean literals", func(t *testing.T) {
		calls = nil
		var resp struct{ Literals *string }
		err := c.Post(`{ literals(x: "a") }`, &resp)
		// The variable true, a *bool, hides the literal that the generated code writes for
		// the arguments false and on, so unmarshaling them fails before the directive runs.
		require.EqualError(t, err, `[{"message":"*bool is not a bool","path":["literals","x"]}]`)
		require.Empty(t, calls)
	})

	t.Run("input object", func(t *testing.T) {
		calls = nil
		var resp struct{ Probed *string }
		require.NoError(t, c.Post(`{ probed(input: {v: "b"}) }`, &resp))
		// The variable asMap hides the fields of the input, so the directive receives the
		// value of its argument asMap as the object.
		require.Equal(t, []string{
			"obj=&m asMap=m it=i",
		}, calls)
	})
}

func deref[T any](p *T) string {
	if p == nil {
		return "nil"
	}
	return fmt.Sprint(*p)
}

// show formats the object that a directive receives, following a pointer to its value.
func show(obj any) string {
	if v := reflect.ValueOf(obj); v.Kind() == reflect.Pointer && !v.IsNil() {
		return "&" + fmt.Sprint(v.Elem().Interface())
	}
	return fmt.Sprint(obj)
}
