package execbehavior

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func TestResolverErrors(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.ErrorProbe = func(ctx context.Context) (*ErrorProbe, error) {
		return &ErrorProbe{Ok: "ok"}, nil
	}
	resolvers.QueryResolver.FailingRoot = func(ctx context.Context) (*string, error) {
		return nil, errors.New("root failed")
	}
	resolvers.ErrorProbeResolver.Failing = func(ctx context.Context, obj *ErrorProbe) (*string, error) {
		return nil, errors.New("failing failed")
	}
	resolvers.ErrorProbeResolver.FailingNonNull = func(ctx context.Context, obj *ErrorProbe) (string, error) {
		return "", errors.New("non-null failed")
	}
	resolvers.ErrorProbeResolver.WithExtensions = func(ctx context.Context, obj *ErrorProbe) (*string, error) {
		return nil, &gqlerror.Error{
			Message:    "with extensions",
			Extensions: map[string]any{"code": "E_PROBE"},
		}
	}
	resolvers.ErrorProbeResolver.Multiple = func(ctx context.Context, obj *ErrorProbe) (*string, error) {
		return nil, gqlerror.List{gqlerror.Errorf("first"), gqlerror.Errorf("second")}
	}
	resolvers.ErrorProbeResolver.ValueAndError = func(ctx context.Context, obj *ErrorProbe) (*string, error) {
		s := "ignored"
		return &s, errors.New("value and error")
	}
	srv := newServer(resolvers, DirectiveRoot{})

	tests := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "nullable field",
			query: `{ errorProbe { ok failing } }`,
			want: `{"data":{"errorProbe":{"ok":"ok","failing":null}},"errors":[
				{"message":"failing failed","path":["errorProbe","failing"]}]}`,
		},
		{
			name:  "non-null field nulls the parent",
			query: `{ errorProbe { ok failingNonNull } }`,
			want: `{"data":{"errorProbe":null},"errors":[
				{"message":"non-null failed","path":["errorProbe","failingNonNull"]}]}`,
		},
		{
			name:  "error with extensions",
			query: `{ errorProbe { withExtensions } }`,
			want: `{"data":{"errorProbe":{"withExtensions":null}},"errors":[
				{"message":"with extensions","path":["errorProbe","withExtensions"],"extensions":{"code":"E_PROBE"}}]}`,
		},
		{
			name:  "list of errors",
			query: `{ errorProbe { multiple } }`,
			want: `{"data":{"errorProbe":{"multiple":null}},"errors":[
				{"message":"first","path":["errorProbe","multiple"]},
				{"message":"second","path":["errorProbe","multiple"]}]}`,
		},
		{
			name:  "value returned with an error is dropped",
			query: `{ errorProbe { valueAndError } }`,
			want: `{"data":{"errorProbe":{"valueAndError":null}},"errors":[
				{"message":"value and error","path":["errorProbe","valueAndError"]}]}`,
		},
		{
			name:  "root field",
			query: `{ failingRoot errorProbe { ok } }`,
			want: `{"data":{"failingRoot":null,"errorProbe":{"ok":"ok"}},"errors":[
				{"message":"root failed","path":["failingRoot"]}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.JSONEq(t, tt.want, post(t, srv, tt.query))
		})
	}
}
