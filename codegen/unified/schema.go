package unified

import (
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/vektah/gqlparser/ast"
	"golang.org/x/tools/go/loader"
)

// Schema is the result of merging the GraphQL Schema with the existing go code
type Schema struct {
	SchemaFilename config.SchemaFilenames
	Config         *config.Config
	Schema         *ast.Schema
	SchemaStr      map[string]string
	Program        *loader.Program
	Directives     map[string]*Directive
	NamedTypes     NamedTypes
	Objects        Objects
	Inputs         Objects
	Interfaces     []*Interface
	Enums          []Enum
}
