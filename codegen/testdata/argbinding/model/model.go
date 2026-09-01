package model

// Obj binds the GraphQL field Obj.field to a Go method whose id parameter is an
// int64, while the default binding for the GraphQL Int type is int.
type Obj struct{}

func (o *Obj) Field(id int64) (string, error) {
	return "", nil
}
