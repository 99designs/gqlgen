package execbehavior

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInputCoercion(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.Coerce = func(ctx context.Context, input *CoerceInput, numbers []int, nested [][]int) (*string, error) {
		var items []string
		if input != nil {
			for _, item := range input.Items {
				items = append(items, string(item.Value))
			}
		}
		var inputNumbers []int
		if input != nil {
			inputNumbers = input.Numbers
		}
		s := fmt.Sprintf(
			"numbers=%v nested=%v input.numbers=%v input.items=%v",
			numbers,
			nested,
			inputNumbers,
			items,
		)
		return &s, nil
	}
	srv := newServer(resolvers, DirectiveRoot{})

	tests := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "lists",
			query: `{ coerce(numbers: [1, 2], nested: [[1], [2, 3]], input: { numbers: [4], items: [{ value: "a" }, { value: "b" }] }) }`,
			want:  `{"data":{"coerce":"numbers=[1 2] nested=[[1] [2 3]] input.numbers=[4] input.items=[a b]"}}`,
		},
		{
			name:  "single values become lists",
			query: `{ coerce(numbers: 1, nested: 2, input: { numbers: 3, items: { value: "a" } }) }`,
			want:  `{"data":{"coerce":"numbers=[1] nested=[[2]] input.numbers=[3] input.items=[a]"}}`,
		},
		{
			name:  "invalid value in a list of inputs",
			query: `{ coerce(input: { items: [{ value: "a" }, { value: "bad" }] }) }`,
			want: `{"data":{"coerce":null},"errors":[
				{"message":"strict rejects this value","path":["coerce","input","items",1,"value"]}]}`,
		},
		{
			name:  "invalid value in a single input coerced to a list",
			query: `{ coerce(input: { items: { value: "bad" } }) }`,
			want: `{"data":{"coerce":null},"errors":[
				{"message":"strict rejects this value","path":["coerce","input","items",0,"value"]}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.JSONEq(t, tt.want, post(t, srv, tt.query))
		})
	}
}
