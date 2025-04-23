package binding

import (
	"context"
	"fmt"
	"io"

	"github.com/99designs/gqlgen/graphql"
)

type Number int

func (e *Number) UnmarshalGQL(v any) error {
	num, err := graphql.UnmarshalInt(v)
	if err != nil {
		return err
	}
	*e = Number(num)
	return nil
}

func (e Number) MarshalGQL(w io.Writer) error {
	fmt.Fprint(w, e)
	return nil
}

type ContextNumber int

func (e *ContextNumber) UnmarshalGQLContext(ctx context.Context, v any) error {
	num, err := graphql.UnmarshalInt(v)
	if err != nil {
		return err
	}
	*e = Number(num)
	return nil
}

func (e ContextNumber) MarshalGQLContext(_ context.Context, w io.Writer) error {
	fmt.Fprint(w, e)
	return nil
}
