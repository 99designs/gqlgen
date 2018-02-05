package jsonw

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJsonWriter(t *testing.T) {
	b := &bytes.Buffer{}
	w := New(b)
	w.BeginObject()
	w.ObjectKey("test")
	w.Int(10)
	w.ObjectKey("array")
	w.BeginArray()
	w.Int(1)
	w.String("2")
	w.True()
	w.False()
	w.Null()
	w.Float64(1.3)
	w.Bool(true)
	w.EndArray()
	w.ObjectKey("emptyArray")

	w.BeginArray()
	w.EndArray()

	w.ObjectKey("child")
	w.BeginObject()
	w.ObjectKey("child")

	w.BeginObject()
	w.ObjectKey("child")
	w.Null()
	w.EndObject()

	w.EndObject()

	w.EndObject()

	require.Equal(t, `{"test":10,"array":[1,"2",true,false,null,1.300000,true],"emptyArray":[],"child":{"child":{"child":null}}}`, b.String())
}
