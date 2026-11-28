package nopanichandler

// PanicObject panics when its boom field is resolved.
type PanicObject struct {
	Name string
}

// Boom is a method field without a context, so it is resolved in place.
func (PanicObject) Boom() *string {
	panic("boom")
}
