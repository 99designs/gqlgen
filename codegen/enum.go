package codegen

type Enum struct {
	Definition  *TypeDefinition
	Description string
	Values      []EnumValue
}

type EnumValue struct {
	Name        string
	Description string
}
