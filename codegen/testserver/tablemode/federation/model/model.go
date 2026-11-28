// Package model holds the Go types of the federation schema of the tablemode test server,
// which both generated packages bind.
package model

type Hello struct {
	Name     string
	Greeting string
}

func (Hello) IsEntity() {}

type World struct {
	Foo   string
	Bar   int
	Hello *Hello
}

func (World) IsEntity() {}

type Planet struct {
	Name     string
	Diameter int
}

func (Planet) IsEntity() {}

// Size is computed from the diameter, which the representation gives.
func (p Planet) Size() int { return p.Diameter * 10 }

type MultiHello struct {
	Name string
}

func (MultiHello) IsEntity() {}

type MultiHelloByNamesInput struct {
	Name string
}

type Product struct {
	ID       string
	Category string
	Price    *int
	Display  string
}

func (Product) IsEntity() {}

type ProductByIDsInput struct {
	ID       string
	Category string
}

type Node interface {
	IsNode()
	IsEntity()
}

type Item struct {
	ID    string
	Label *string
}

func (Item) IsNode()   {}
func (Item) IsEntity() {}
