package codegen

type Enum struct {
	Definition *TypeDefinition
	Values     []EnumValue
	InTypemap  bool
}

type EnumValue struct {
	Name        string
	Description string
}
