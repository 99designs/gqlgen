package singlefile

type PtrToAnyContainer struct {
	PtrToAny *any
}

func (c *PtrToAnyContainer) Content() *any {
	return c.PtrToAny
}
