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

// The generated code declares the arguments of a directive as variables beside the object
// it passes to the directive, the next resolver, its receiver, the packages it calls and the
// values it writes for the other arguments, and DirectiveRoot declares them as parameters
// beside ctx, obj, next, res and err. Arguments with the names of those must not hide them.
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
				objArg, rawArgs, asMap, it, ctxArg, errArg, ec, nextArg, resArg, directive0,
				errorsArg, anyArg, nilArg *string, trueArg, falseArg *bool, fooBar, fooBarArg *string,
				on *bool, list []*string,
			) (any, error) {
				got := []string{"obj=" + show(obj)}
				for _, a := range []struct {
					name  string
					value string
				}{
					{"obj", deref(objArg)}, {"rawArgs", deref(rawArgs)}, {"asMap", deref(asMap)},
					{"it", deref(it)}, {"ctx", deref(ctxArg)}, {"err", deref(errArg)},
					{"ec", deref(ec)}, {"next", deref(nextArg)}, {"res", deref(resArg)},
					{"directive0", deref(directive0)}, {"errors", deref(errorsArg)},
					{"any", deref(anyArg)}, {"nil", deref(nilArg)}, {"true", deref(trueArg)},
					{"false", deref(falseArg)}, {"foo_bar", deref(fooBar)}, {"fooBar", deref(fooBarArg)},
					{"on", deref(on)},
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
			"obj=<nil> any=a nil=n list=[l]",
		}, calls)
	})

	t.Run("argument", func(t *testing.T) {
		calls = nil
		var resp struct{ Probed *string }
		require.NoError(t, c.Post(`{ probed(x: "a") }`, &resp))
		require.Equal(t, []string{
			"obj=map[x:a] obj=o rawArgs=r ctx=c err=e ec=ec next=nx res=rs directive0=d0 " +
				"errors=er foo_bar=fb fooBar=fB",
		}, calls)
	})

	t.Run("boolean literals", func(t *testing.T) {
		calls = nil
		var resp struct{ Literals *string }
		require.NoError(t, c.Post(`{ literals(x: "a") }`, &resp))
		require.Equal(t, []string{
			"obj=map[x:a] true=false false=true on=true",
		}, calls)
	})

	t.Run("input object", func(t *testing.T) {
		calls = nil
		var resp struct{ Probed *string }
		require.NoError(t, c.Post(`{ probed(input: {v: "b"}) }`, &resp))
		require.Equal(t, []string{
			"obj=map[v:b] asMap=m it=i",
		}, calls)
	})
}

func deref[T any](p *T) string {
	if p == nil {
		return "nil"
	}
	return fmt.Sprint(*p)
}

// show formats the object that a directive receives, following a pointer to its value, so
// that a directive that receives an argument in place of the object shows its value.
func show(obj any) string {
	if v := reflect.ValueOf(obj); v.Kind() == reflect.Pointer && !v.IsNil() {
		return "&" + fmt.Sprint(v.Elem().Interface())
	}
	return fmt.Sprint(obj)
}
