package fedruntime

// GqlgenService is the service object that the
// generated.go file will return for the _service
// query
type GqlgenService struct {
	SDL string `json:"sdl"`
}

// Everything with a @key implements this
type GqlgenEntity interface {
	IsGqlgenEntity()
}
