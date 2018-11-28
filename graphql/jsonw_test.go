package graphql

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJsonWriter(t *testing.T) {
	obj := &OrderedMap{}
	obj.Add("test", MarshalInt(10))

	obj.Add("array", &Array{
		MarshalInt(1),
		MarshalString("2"),
		MarshalBoolean(true),
		False,
		Null,
		MarshalFloat(1.3),
		True,
	})

	obj.Add("emptyArray", &Array{})

	child2 := &OrderedMap{}
	child2.Add("child", Null)

	child1 := &OrderedMap{}
	child1.Add("child", child2)

	obj.Add("child", child1)

	b := &bytes.Buffer{}
	obj.MarshalGQL(b)

	require.Equal(t, `{"test":10,"array":[1,"2",true,false,null,1.3,true],"emptyArray":[],"child":{"child":{"child":null}}}`, b.String())
}
