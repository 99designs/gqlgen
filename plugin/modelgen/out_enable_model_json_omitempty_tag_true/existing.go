package out_enable_model_json_omitempty_tag_true

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
