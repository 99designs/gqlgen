package followschema

import "github.com/99designs/gqlgen/codegen/testserver/followschema/otherpkg"

type WrappedScalar = otherpkg.Scalar
type WrappedStruct otherpkg.Struct
type WrappedMap otherpkg.Map
type WrappedSlice otherpkg.Slice
