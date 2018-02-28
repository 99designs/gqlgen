package codegen

type Model struct {
	*NamedType

	Fields []ModelField
}

type ModelField struct {
	*Type

	GoVarName string
	GoFKName  string
	GoFKType  string
}
