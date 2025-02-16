package deferexample

import "sync"

type Resolver struct {
	mu    sync.RWMutex
	todos []*Todo
}
