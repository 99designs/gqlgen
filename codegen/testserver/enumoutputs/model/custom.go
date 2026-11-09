package model

import (
	"context"
	"errors"
	"io"
	"strconv"
)

type Bound int

const (
	One Bound = 1
	Two Bound = 2
)

type Custom string

func (v Custom) MarshalGQL(w io.Writer) { io.WriteString(w, strconv.Quote(string(v))) }
func (v *Custom) UnmarshalGQL(input any) error {
	s, ok := input.(string)
	if !ok {
		return errors.New("expected string")
	}
	*v = Custom(s)
	return nil
}

type Contextual string

func (v Contextual) MarshalGQLContext(ctx context.Context, w io.Writer) error {
	if v == "ERROR" {
		io.WriteString(w, `"partial`)
		return errors.New("context enum failed")
	}
	_, err := io.WriteString(w, strconv.Quote(string(v)))
	return err
}

func (v *Contextual) UnmarshalGQLContext(ctx context.Context, input any) error {
	s, ok := input.(string)
	if !ok {
		return errors.New("expected string")
	}
	*v = Contextual(s)
	return nil
}
