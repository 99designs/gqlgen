package codegen

type Model struct {
	Definition  *TypeDefinition
	Description string
	Fields      []ModelField
	Implements  []*TypeDefinition
}

type ModelField struct {
	*TypeReference
	GQLName     string
	GoFieldName string
	GoFKName    string
	GoFKType    string
	Description string
}
