package model

import (
	"github.com/99designs/gqlgen/example/scalars/external"
)

type Address struct {
	ID       external.ObjectID `json:"id"`
	Location *Point            `json:"location"`
}
