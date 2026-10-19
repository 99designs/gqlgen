package singlefile

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

// unknownShape satisfies the Go Shape interface without being one of the
// schema's implementations, so the generated Shape marshaller panics on it.
type unknownShape struct{}

func (unknownShape) Area() float64 { return 0 }
func (unknownShape) isShape()      {}

// A list element whose marshaller panics fails like an element whose resolver
// returned an error: shapes is [Shape], so the element reports null in place,
// one error names its index, and the other elements are kept.
func TestListElementPanic(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		shapes     []Shape
		wantShapes []any
		wantPath   string
	}{
		"only element panics": {
			shapes:     []Shape{unknownShape{}},
			wantShapes: []any{nil},
			wantPath:   `["shapes",0]`,
		},
		"elements around the panicking one are kept": {
			shapes:     []Shape{&Rectangle{Length: 2, Width: 3}, unknownShape{}},
			wantShapes: []any{map[string]any{"area": 6.0}, nil},
			wantPath:   `["shapes",1]`,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resolvers := &Stub{}
			resolvers.QueryResolver.Shapes = func(context.Context) ([]Shape, error) {
				return tc.shapes, nil
			}
			srv := handler.New(NewExecutableSchema(Config{Resolvers: resolvers}))
			srv.AddTransport(transport.POST{})

			var resp struct {
				Shapes []any
			}
			err := client.New(srv).Post(`query { shapes { area } }`, &resp)

			var rawErr client.RawJsonError
			require.ErrorAs(t, err, &rawErr, "the recovered panic is reported")
			assert.JSONEq(t,
				`[{"message":"internal system error","path":`+tc.wantPath+`}]`,
				string(rawErr.RawMessage))
			assert.Equal(t, tc.wantShapes, resp.Shapes)
		})
	}
}
