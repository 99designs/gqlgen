package codegen

type Interface struct {
	*NamedType

	Implementors []*NamedType
}
