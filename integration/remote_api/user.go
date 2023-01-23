package remote_api

import (
	"fmt"
	"io"
	"strconv"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/integration/testomitempty"
)

type User struct {
	Name  string
	Likes []string
}

type DefinedTypeFromBasics struct {
	NewString  testomitempty.NamedString  `json:"newString"`
	NewInt     testomitempty.NamedInt     `json:"newInt"`
	NewInt8    testomitempty.NamedInt8    `json:"newInt8"`
	NewInt16   testomitempty.NamedInt16   `json:"newInt16"`
	NewInt32   testomitempty.NamedInt32   `json:"newInt32"`
	NewInt64   testomitempty.NamedInt64   `json:"newInt64"`
	NewBool    testomitempty.NamedBool    `json:"newBool"`
	NewFloat32 testomitempty.NamedFloat32 `json:"newFloat32"`
	NewFloat64 testomitempty.NamedFloat64 `json:"newFloat64"`
	NewUint    testomitempty.NamedUint    `json:"newUint"`
	NewUint8   testomitempty.NamedUint8   `json:"newUint8"`
	NewUint16  testomitempty.NamedUint16  `json:"newUint16"`
	NewUint32  testomitempty.NamedUint32  `json:"newUint32"`
	NewUint64  testomitempty.NamedUint64  `json:"newUint64"`
	NewID      testomitempty.NamedID      `json:"newID"`
}

// Lets redefine the base Float32 type
func MarshalFloat32(id testomitempty.NamedFloat32) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		io.WriteString(w, strconv.Quote(fmt.Sprintf("=%v=", id)))
	})
}

// And the same for the unmarshaler
func UnmarshalFloat32(v interface{}) (testomitempty.NamedFloat32, error) {
	str, ok := v.(string)
	if !ok {
		return 0, fmt.Errorf("float32 must be Float32")
	}
	i, err := strconv.Atoi(str[1 : len(str)-1])
	return testomitempty.NamedFloat32(i), err
}

// Lets redefine the base Uint64 type
func MarshalUint64(id testomitempty.NamedUint64) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		io.WriteString(w, strconv.Quote(fmt.Sprintf("=%d=", id)))
	})
}

// And the same for the unmarshaler
func UnmarshalUint64(v interface{}) (testomitempty.NamedUint64, error) {
	str, ok := v.(string)
	if !ok {
		return 0, fmt.Errorf("uint64 must be Uint64")
	}
	i, err := strconv.Atoi(str[1 : len(str)-1])
	return testomitempty.NamedUint64(i), err
}

// Lets redefine the base ID type to use an id from an external library
func MarshalID(id testomitempty.NamedID) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		io.WriteString(w, strconv.Quote(fmt.Sprintf("=%d=", id)))
	})
}

// And the same for the unmarshaler
func UnmarshalID(v interface{}) (testomitempty.NamedID, error) {
	str, ok := v.(string)
	if !ok {
		return 0, fmt.Errorf("ids must be strings")
	}
	i, err := strconv.Atoi(str[1 : len(str)-1])
	return testomitempty.NamedID(i), err
}
