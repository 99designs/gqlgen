//go:generate rm -rf generated
//go:generate go run ../../testdata/gqlgen.go

package benchmarking

import (
	"context"
	"errors"
	"github.com/99designs/gqlgen/_examples/benchmarking/generated"
	"github.com/99designs/gqlgen/_examples/benchmarking/models"
	"strings"
)

type Resolver struct {
	input, output string
}

func (r Resolver) Mutation() generated.MutationResolver {
	return &mutationResolver{r}
}

func (r Resolver) Query() generated.QueryResolver {
	return &queryResolver{r}
}

type queryResolver struct{ Resolver }

func (q *queryResolver) TestQueryPerformance(_ context.Context, in models.InputArgument) (*models.OutputType, error) {
	if q.input != in.Value {
		return nil, errors.New("input value does not match expected value")
	}
	return &models.OutputType{Value: q.output}, nil
}

type mutationResolver struct{ Resolver }

func (m *mutationResolver) TestMutationPerformance(_ context.Context, in models.InputArgument) (*models.OutputType, error) {
	if m.input != in.Value {
		return nil, errors.New("input value does not match expected value")
	}
	return &models.OutputType{Value: m.output}, nil
}

func GenRandomString(size int) string {
	return strings.Repeat("a", size)
}

func NewResolver(outputSize int, expectedInput string) *Resolver {
	return &Resolver{output: GenRandomString(outputSize), input: expectedInput}
}
