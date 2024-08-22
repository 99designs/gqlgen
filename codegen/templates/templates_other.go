//go:build !go1.23

package templates

import (
	"fmt"
	"go/types"
)

func TypeIdentifier(t types.Type) string {
	res := ""
	for {
		switch it := t.(type) {
		case *types.Pointer:
			t.Underlying()
			res += "ᚖ"
			t = it.Elem()
		case *types.Slice:
			res += "ᚕ"
			t = it.Elem()
		case *types.Named:
			res += pkgReplacer.Replace(it.Obj().Pkg().Path())
			res += "ᚐ"
			res += it.Obj().Name()
			return res
		case *types.Basic:
			res += it.Name()
			return res
		case *types.Map:
			res += "map"
			return res
		case *types.Interface:
			res += "interface"
			return res
		default:
			panic(fmt.Errorf("unexpected type %T", it))
		}
	}
}
