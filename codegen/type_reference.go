package codegen

import (
	"go/types"
	"strconv"

	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/vektah/gqlparser/ast"
)

// TypeReference represents the type of a field or arg, referencing an underlying TypeDefinition (type, input, scalar)
type TypeReference struct {
	Definition *TypeDefinition

	GoType  types.Type
	ASTType *ast.Type
}

// todo @vektah: This should probably go away, its too easy to conflate gql required vs go pointer
func (t TypeReference) IsPtr() bool {
	_, isPtr := t.GoType.(*types.Pointer)
	return isPtr
}

func (t TypeReference) Unmarshal(result, raw string) string {
	return t.unmarshal(result, raw, t.GoType, 1)
}

func (t TypeReference) unmarshal(result, raw string, destType types.Type, depth int) string {
	switch destType := destType.(type) {
	case *types.Pointer:
		ptr := "ptr" + strconv.Itoa(depth)
		return tpl(`var {{.ptr}} {{.destType | ref }}
			if {{.raw}} != nil {
				{{.next}}
				{{.result}} = &{{.ptr -}}
			}
		`, map[string]interface{}{
			"ptr":      ptr,
			"t":        t,
			"raw":      raw,
			"result":   result,
			"destType": destType.Elem(),
			"next":     t.unmarshal(ptr, raw, destType.Elem(), depth+1),
		})

	case *types.Slice:
		var rawIf = "rawIf" + strconv.Itoa(depth)
		var index = "idx" + strconv.Itoa(depth)

		return tpl(`var {{.rawSlice}} []interface{}
			if {{.raw}} != nil {
				if tmp1, ok := {{.raw}}.([]interface{}); ok {
					{{.rawSlice}} = tmp1
				} else {
					{{.rawSlice}} = []interface{}{ {{.raw}} }
				}
			}
			{{.result}} = make({{.destType | ref}}, len({{.rawSlice}}))
			for {{.index}} := range {{.rawSlice}} {
				{{ .next -}}
			}`, map[string]interface{}{
			"raw":      raw,
			"rawSlice": rawIf,
			"index":    index,
			"result":   result,
			"destType": destType,
			"next":     t.unmarshal(result+"["+index+"]", rawIf+"["+index+"]", destType.Elem(), depth+1),
		})
	}

	realResult := result

	return tpl(`
			{{- if eq (.t.Definition.GoType | ref) "map[string]interface{}" }}
				{{- .result }} = {{.raw}}.(map[string]interface{})
			{{- else if .t.Definition.Unmarshaler }}
				{{- .result }}, err = {{ .t.Definition.Unmarshaler | call }}({{.raw}})
			{{- else -}}
				err = (&{{.result}}).UnmarshalGQL({{.raw}})
			{{- end }}`, map[string]interface{}{
		"realResult": realResult,
		"result":     result,
		"raw":        raw,
		"t":          t,
	})
}

func (t TypeReference) Middleware(result, raw string) string {
	return t.middleware(result, raw, t.GoType, 1)
}

func (t TypeReference) middleware(result, raw string, destType types.Type, depth int) string {
	switch destType := destType.(type) {
	case *types.Pointer:
		switch destType.Elem().(type) {
		case *types.Pointer, *types.Slice:
			return tpl(`if {{.raw}} != nil {
				{{.next}}
			}`, map[string]interface{}{
				"t":      t,
				"raw":    raw,
				"result": result,
				"next":   t.middleware(result, raw, destType.Elem(), depth+1),
			})
		default:
			return tpl(`
			if {{.raw}} != nil {
				var err error
				{{.result}}, err = e.{{ .t.Definition.GQLType }}Middleware(ctx, {{.raw}})
				if err != nil {
					return nil, err
				}
			}`, map[string]interface{}{
				"result": result,
				"raw":    raw,
				"t":      t,
			})
		}

	case *types.Slice:
		var index = "idx" + strconv.Itoa(depth)

		return tpl(`for {{.index}} := range {{.raw}} {
				{{ .next -}}
			}`, map[string]interface{}{
			"raw":    raw,
			"index":  index,
			"result": result,
			"next":   t.middleware(result+"["+index+"]", raw+"["+index+"]", destType.Elem(), depth+1),
		})
	}

	ptr := "m" + t.Definition.GQLType + strconv.Itoa(depth)
	return tpl(`
			{{.ptr}}, err := e.{{ .t.Definition.GQLType }}Middleware(ctx, &{{.raw}})
				if err != nil {	
					return nil, err
			}
			{{ .result }} = *{{.ptr}}`, map[string]interface{}{
		"result": result,
		"raw":    raw,
		"ptr":    ptr,
		"t":      t,
	})
}

func (t TypeReference) Marshal(val string) string {
	if t.Definition.Marshaler != nil {
		return "return " + templates.Call(t.Definition.Marshaler) + "(" + val + ")"
	}

	return "return " + val
}
