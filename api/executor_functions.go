//go:build !gqlgen_table

package api

import (
	"github.com/99designs/gqlgen/codegen"
)

// The executor is generated in the functions mode, unless gqlgen is built with the
// gqlgen_table tag: see executor_table.go.

// prepareExecutor returns the function that generates the executor from data, after the
// plugins generated their code: incrementally when it is given the options of the
// incremental generation.
func prepareExecutor(data *codegen.Data) (func(*codegen.IncrementalOptions) error, error) {
	return func(incrementalOpts *codegen.IncrementalOptions) error {
		if incrementalOpts != nil {
			return codegen.GenerateCodeIncremental(data, *incrementalOpts)
		}
		return codegen.GenerateCode(data)
	}, nil
}
