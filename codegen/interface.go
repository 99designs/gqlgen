package codegen

type Interface struct {
	*TypeDefinition

	Implementors []InterfaceImplementor
}

type InterfaceImplementor struct {
	ValueReceiver bool

	*TypeDefinition
}
