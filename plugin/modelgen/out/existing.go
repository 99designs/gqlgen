package out

type ExistingType struct {
	Name     *string              `json:"name,omitempty"`
	Enum     *ExistingEnum        `json:"enum,omitempty"`
	Int      ExistingInterface    `json:"int,omitempty"`
	Existing *MissingTypeNullable `json:"existing,omitempty"`
}

type ExistingModel struct {
	Name string
	Enum ExistingEnum
	Int  ExistingInterface
}

type ExistingInput struct {
	Name string
	Enum ExistingEnum
	Int  ExistingInterface
}

type ExistingEnum string

type ExistingInterface interface {
	IsExistingInterface()
}

type ExistingUnion interface {
	IsExistingUnion()
}
