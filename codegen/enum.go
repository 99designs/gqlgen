package codegen

type Enum struct {
	*TypeDefinition
	Description string
	Values      []EnumValue
}

type EnumValue struct {
	Name        string
	Description string
}
