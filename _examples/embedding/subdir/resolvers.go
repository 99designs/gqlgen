//go:generate go run ../../../testdata/gqlgen.go -config cfgdir/generate_in_subdir.yml
//go:generate go run ../../../testdata/gqlgen.go -config cfgdir/generate_in_gendir.yml

package subdir

import (
	"context"

	"github.com/99designs/gqlgen/_examples/embedding/subdir/gendir"
)

type Resolver struct{ *Resolver }

func (q *Resolver) Query() QueryResolver {
	return q
}
func (q *Resolver) InSchemadir(ctx context.Context) (string, error) {
	return "example", nil
}
func (q *Resolver) Parentdir(ctx context.Context) (string, error) {
	return "example", nil
}
func (q *Resolver) Subdir(ctx context.Context) (string, error) {
	return "example", nil
}

type GendirResolver struct{ *Resolver }

func (q *GendirResolver) Query() gendir.QueryResolver {
	return &Resolver{}
}
