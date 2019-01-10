package codegen

type Interface struct {
	Definition   *TypeDefinition
	Implementors []InterfaceImplementor
	InTypemap    bool
}

type InterfaceImplementor struct {
	ValueReceiver bool
	Definition    *TypeDefinition
}
