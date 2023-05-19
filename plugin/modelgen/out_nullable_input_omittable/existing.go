package out_nullable_input_omittable

import (
	"github.com/99designs/gqlgen/graphql"
)

type ExistingType struct {
	Name     *string              `json:"name"`
	Enum     *ExistingEnum        `json:"enum"`
	Int      ExistingInterface    `json:"int"`
	Existing *MissingTypeNullable `json:"existing"`
}

type ExistingModel struct {
	Name string
	Enum ExistingEnum
	Int  ExistingInterface
}

type ExistingInput struct {
	Name graphql.Omittable[string]
	Enum graphql.Omittable[ExistingEnum]
	Int  graphql.Omittable[ExistingInterface]
}

type ExistingEnum string

type ExistingInterface interface {
	IsExistingInterface()
}

type ExistingUnion interface {
	IsExistingUnion()
}
