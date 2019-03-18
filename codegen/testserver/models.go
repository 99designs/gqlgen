package testserver

import (
	"context"
	"fmt"
	"io"
)

type ForcedResolver struct {
	Field Circle
}

type ModelMethods struct {
}

func (m ModelMethods) NoContext() bool {
	return true
}

func (m ModelMethods) WithContext(_ context.Context) bool {
	return true
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

type MarshalPanic string

func (m *MarshalPanic) UnmarshalGQL(v interface{}) error {
	panic("BOOM")
}

func (m MarshalPanic) MarshalGQL(w io.Writer) {
	panic("BOOM")
}

type Panics struct {
}

func (p *Panics) FieldFuncMarshal(ctx context.Context, u []MarshalPanic) []MarshalPanic {
	return []MarshalPanic{MarshalPanic("aa"), MarshalPanic("bb")}
}

type Autobind struct {
	Int   int
	Int32 int32
	Int64 int64

	IdStr string
	IdInt int
}

type OverlappingFields struct {
	Foo    int
	NewFoo int
}

type FallbackToStringEncoding string

const (
	FallbackToStringEncodingA FallbackToStringEncoding = "A"
	FallbackToStringEncodingB FallbackToStringEncoding = "B"
	FallbackToStringEncodingC FallbackToStringEncoding = "C"
)
