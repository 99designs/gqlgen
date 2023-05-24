package singlefile

type PtrToAnyContainer struct {
	PtrToAny *any
}

func (c *PtrToAnyContainer) Binding() *any {
	return c.PtrToAny
}
