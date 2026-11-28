package execbehavior

import (
	"io"
	"strconv"
)

type Animal interface {
	IsAnimal()
}

type Pet interface {
	IsPet()
}

// Dog implements the interfaces with value receivers, so both Dog and *Dog are
// animals.
type Dog struct {
	Name  string
	Barks bool
}

func (Dog) IsAnimal() {}
func (Dog) IsPet()    {}

// Cat implements the interfaces with pointer receivers, so only *Cat is an animal.
type Cat struct {
	Name  string
	Lives int
}

func (*Cat) IsAnimal() {}
func (*Cat) IsPet()    {}

// Robot is an animal that is not in the schema and marshals itself.
type Robot struct{ Model string }

func (Robot) IsAnimal() {}

func (r Robot) MarshalGQL(w io.Writer) {
	_, _ = io.WriteString(w, strconv.Quote("robot "+r.Model))
}

// Ghost is an animal that is not in the schema and cannot be marshaled.
type Ghost struct{}

func (Ghost) IsAnimal() {}
