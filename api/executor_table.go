//go:build gqlgen_table

package api

import (
	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/internal/tablecodegen"
)

// Table mode is experimental: only gqlgen built with the gqlgen_table tag links its
// generator, and generates the executor in it, as in
//
//	go run -tags gqlgen_table github.com/99designs/gqlgen
//
// gqlgen built without the tag generates in the functions mode: see executor_functions.go.

// prepareExecutor builds the data of table mode from data, before the plugins generate
// their code, so that it can tell what they changed, and returns the function that
// generates the executor from it afterwards: incrementally when it is given the options
// of the incremental generation.
func prepareExecutor(data *codegen.Data) (func(*codegen.IncrementalOptions) error, error) {
	tableData, err := tablecodegen.Build(data)
	if err != nil {
		return nil, err
	}
	return func(incrementalOpts *codegen.IncrementalOptions) error {
		if incrementalOpts != nil {
			return tablecodegen.GenerateCodeIncremental(tableData, *incrementalOpts)
		}
		return tablecodegen.GenerateCode(tableData)
	}, nil
}
