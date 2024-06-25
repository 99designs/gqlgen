package benchmarking

import (
	"fmt"
	"github.com/99designs/gqlgen/_examples/benchmarking/models"
	gqlclient "github.com/99designs/gqlgen/client"
	"testing"

	"github.com/99designs/gqlgen/_examples/benchmarking/generated"
	"github.com/99designs/gqlgen/graphql/handler"
)

func BenchmarkQueriesOfVariableSizes(bmark *testing.B) {
	for _, testCase := range []struct {
		inputSize  int
		outputSize int
	}{
		{inputSize: 100000, outputSize: 100000},
		{inputSize: 1000000, outputSize: 1000000},
		{inputSize: 10000000, outputSize: 1000000},
		{inputSize: 10000000, outputSize: 10000000},
		{inputSize: 100000000, outputSize: 100000000},
	} {
		bmark.Run(fmt.Sprintf("input size: %d output size %d", testCase.inputSize, testCase.outputSize), func(b *testing.B) {
			input := generateStringOfSize(testCase.inputSize)

			result := &struct {
				Out models.OutputType `json:"testQueryPerformance"`
			}{}
			gql := gqlclient.New(handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: NewResolver(testCase.outputSize, input)})))
			q := `query ($arg: InputArgument!) {
			testQueryPerformance(in: $arg) {
				value
			}
		}`

			b.ReportAllocs()
			b.ResetTimer()

			for j := 0; j < b.N; j++ {
				gql.MustPost(q, result, gqlclient.Var("arg", models.InputArgument{Value: input}))
				if len(result.Out.Value) != testCase.outputSize {
					b.Fatalf("Unexpected output size: %d", len(result.Out.Value))
				}
			}
		})
	}
}
