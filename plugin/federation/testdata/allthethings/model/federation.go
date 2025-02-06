package model

type _FieldSet string //nolint:deadcode,unused

type Hello struct {
	Name      string
	Secondary string
}

func (Hello) IsEntity() {}

type World struct {
	Foo string
	Bar int
}

func (World) IsEntity() {}

type MultiHelloMultiKey struct {
	Name string
	Key2 int
}

func (MultiHelloMultiKey) IsEntity() {}

type ExternalExtension struct {
	UPC     string
	Reviews []*World
}

func (ExternalExtension) IsEntity() {}

type NestedKey struct {
	ID    string
	Hello *Hello
}

func (NestedKey) IsEntity() {}

type MoreNesting struct {
	ID    string
	World *World
}

func (MoreNesting) IsEntity() {}

type VeryNestedKey struct {
	ID     string
	Hello  *Hello
	World  *World
	Nested *NestedKey
	More   *MoreNesting
}

func (VeryNestedKey) IsEntity() {}

type MultiHelloMultiKeyByNamesInput struct {
	Names []string `json:"Names"`
}

type MultiHelloMultiKeyByKey2sInput struct {
	Key2s []string `json:"Key2s"`
}
