// Package model holds the Go types of the batchfs schema of the tablemode test server,
// which both generated packages bind.
package model

type Animal interface {
	IsAnimal()
}

type Dog struct {
	ID string
}

func (Dog) IsAnimal() {}

type Cat struct {
	ID string
}

func (Cat) IsAnimal() {}
