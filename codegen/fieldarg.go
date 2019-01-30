package codegen

type FieldArgument struct {
	*TypeReference

	GQLName    string      // The name of the argument in graphql
	GoVarName  string      // The name of the var in go
	Object     *Object     // A link back to the parent object
	Default    interface{} // The default value
	Directives []*Directive
	Value      interface{} // value set in Data
}

func (f *FieldArgument) Stream() bool {
	return f.Object != nil && f.Object.Stream
}
