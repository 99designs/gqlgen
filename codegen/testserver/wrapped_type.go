package testserver

import (
	"fmt"
	"io"
	"strconv"

	"github.com/99designs/gqlgen/codegen/testserver/otherpkg"
	"github.com/99designs/gqlgen/graphql"
)

type WrappedScalar otherpkg.Scalar
type WrappedStruct otherpkg.Struct
type WrappedMap otherpkg.Map
type WrappedSlice otherpkg.Slice

func (e *WrappedScalar) UnmarshalGQL(v interface{}) error {
	s, err := graphql.UnmarshalString(v)
	if err != nil {
		return err
	}
	*e = WrappedScalar(s)
	return nil
}

func (e WrappedScalar) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(string(e)))
}
