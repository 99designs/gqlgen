package todo

import (
	"context"
	"fmt"
	"io"

	"github.com/99designs/gqlgen/graphql"
)

type Ownable interface {
	Owner() *User
}

type Todo struct {
	ID    int
	Text  string
	Done  bool
	owner *User
}

func (t Todo) Owner() *User {
	return t.owner
}

type User struct {
	ID   int
	Name string
}

type Number int

func (e *Number) UnmarshalGQLContext(ctx context.Context, v any) error {
	num, err := graphql.UnmarshalInt(v)
	if err != nil {
		return err
	}
	*e = Number(num)
	return nil
}

func (e Number) MarshalGQLContext(_ context.Context, w io.Writer) error {
	fmt.Fprint(w, e)
	return nil
}
