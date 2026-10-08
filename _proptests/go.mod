// Property tests live in their own module so that pgregory.net/rapid stays out
// of the dependency graph of github.com/99designs/gqlgen itself. Anyone
// importing gqlgen gets neither rapid nor its transitive dependencies, which is
// the same reason _examples is a separate module.
//
// The leading underscore keeps the Go tool from walking into this directory
// from the parent module, so it needs its own invocation: see the proptests
// target in CI.
module github.com/99designs/gqlgen/_proptests

go 1.26.0

replace github.com/99designs/gqlgen => ../

require (
	github.com/99designs/gqlgen v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.6.0
	github.com/stretchr/testify v1.12.1
	pgregory.net/rapid v1.3.0
)

require (
	github.com/sosodev/duration v1.4.0 // indirect
	github.com/vektah/gqlparser/v2 v2.5.62 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/sync v0.23.0 // indirect
)
