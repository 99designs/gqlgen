package codegen

type Model struct {
	*NamedType

	Fields []ModelField
}

type ModelField struct {
	*Type
	GQLName     string
	GoFieldName string
	GoFKName    string
	GoFKType    string
}
