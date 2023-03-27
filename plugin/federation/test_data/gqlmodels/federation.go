package model

type _FieldSet string //nolint:deadcode,unused

type Hello struct {
	Name      string
	Secondary string
}

func (Hello) IsEntity() {}

type ExternalExtensionByUpcsInput struct {
	Upc string
}

func (ExternalExtensionByUpcsInput) IsEntity() {}

type ExternalExtension struct {
	Upc string
}

func (ExternalExtension) IsEntity() {}
