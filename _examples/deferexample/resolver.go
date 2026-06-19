//go:generate go run ../../testdata/gqlgen.go
package deferexample

import "sync"

type Resolver struct {
	mu    sync.RWMutex
	todos []*Todo
}
