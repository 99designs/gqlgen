package testserver

import "fmt"

type ForcedResolver struct {
	Field Circle
}

type Error struct {
	ID string
}

func (Error) ErrorOnRequiredField() (string, error) {
	return "", fmt.Errorf("boom")
}

func (Error) ErrorOnNonRequiredField() (string, error) {
	return "", fmt.Errorf("boom")
}

func (Error) NilOnRequiredField() *string {
	return nil
}

type EmbeddedPointerModel struct {
	*EmbeddedPointer
	ID string
}

type EmbeddedPointer struct {
	Title string
}
