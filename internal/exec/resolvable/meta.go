package resolvable

import (
	"reflect"

	"github.com/neelance/graphql-go/internal/schema"
	"github.com/neelance/graphql-go/introspection"
)

var MetaSchema *Object
var MetaType *Object

func init() {
	var err error
	b := newBuilder(schema.Meta)

	metaSchema := schema.Meta.Types["__Schema"].(*schema.Object)
	MetaSchema, err = b.makeObjectExec(metaSchema.Name, metaSchema.Fields, nil, false, reflect.TypeOf(&introspection.Schema{}))
	if err != nil {
		panic(err)
	}

	metaType := schema.Meta.Types["__Type"].(*schema.Object)
	MetaType, err = b.makeObjectExec(metaType.Name, metaType.Fields, nil, false, reflect.TypeOf(&introspection.Type{}))
	if err != nil {
		panic(err)
	}

	if err := b.finish(); err != nil {
		panic(err)
	}
}
