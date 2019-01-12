package codegen

import (
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/vektah/gqlparser/ast"
)

// Schema is the result of merging the GraphQL Schema with the existing go code
type Schema struct {
	Config     *config.Config
	Schema     *ast.Schema
	SchemaStr  map[string]string
	Directives map[string]*Directive
	Objects    Objects
	Inputs     Objects
	Interfaces []*Interface
	Enums      []Enum
}

func NewSchema(cfg *config.Config) (*Schema, error) {
	return buildSchema(cfg)
}
