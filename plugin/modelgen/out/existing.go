package out

type ExistingModel struct {
	Name string
	Enum ExistingEnum
	Int  ExistingInterface
}

type ExistingInput struct {
	Name string
	Enum ExistingEnum
	Int  ExistingInterface
}

type ExistingEnum string

type ExistingInterface interface {
	IsExistingInterface()
}

type ExistingUnion interface {
	IsExistingUnion()
}
