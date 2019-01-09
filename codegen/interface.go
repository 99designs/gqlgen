package codegen

type Interface struct {
	Definition   *TypeDefinition
	Implementors []InterfaceImplementor
}

type InterfaceImplementor struct {
	ValueReceiver bool
	Definition    *TypeDefinition
}
