package codegen

import (
	"go/types"
	"strconv"
	"strings"

	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/vektah/gqlparser/ast"
)

// TypeReference represents the type of a field or arg, referencing an underlying TypeDefinition (type, input, scalar)
type TypeReference struct {
	Definition *TypeDefinition

	Modifiers []string
	ASTType   *ast.Type
}

func (t TypeReference) Signature() string {
	return strings.Join(t.Modifiers, "") + templates.CurrentImports.LookupType(t.Definition.GoType)
}

func (t TypeReference) FullSignature() string {
	return strings.Join(t.Modifiers, "") + types.TypeString(t.Definition.GoType, nil)
}

func (t TypeReference) IsPtr() bool {
	return len(t.Modifiers) > 0 && t.Modifiers[0] == modPtr
}

func (t *TypeReference) StripPtr() {
	if !t.IsPtr() {
		return
	}
	t.Modifiers = t.Modifiers[0 : len(t.Modifiers)-1]
}

func (t TypeReference) IsSlice() bool {
	return len(t.Modifiers) > 0 && t.Modifiers[0] == modList ||
		len(t.Modifiers) > 1 && t.Modifiers[0] == modPtr && t.Modifiers[1] == modList
}

func (t TypeReference) Unmarshal(result, raw string) string {
	return t.unmarshal(result, raw, t.Modifiers, 1)
}

func (t TypeReference) unmarshal(result, raw string, remainingMods []string, depth int) string {
	switch {
	case len(remainingMods) > 0 && remainingMods[0] == modPtr:
		ptr := "ptr" + strconv.Itoa(depth)
		return tpl(`var {{.ptr}} {{.mods}}{{.t.Definition.GoType | ref }}
			if {{.raw}} != nil {
				{{.next}}
				{{.result}} = &{{.ptr -}}
			}
		`, map[string]interface{}{
			"ptr":    ptr,
			"t":      t,
			"raw":    raw,
			"result": result,
			"mods":   strings.Join(remainingMods[1:], ""),
			"next":   t.unmarshal(ptr, raw, remainingMods[1:], depth+1),
		})

	case len(remainingMods) > 0 && remainingMods[0] == modList:
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
			{{.result}} = make({{.type}}, len({{.rawSlice}}))
			for {{.index}} := range {{.rawSlice}} {
				{{ .next -}}
			}`, map[string]interface{}{
			"raw":      raw,
			"rawSlice": rawIf,
			"index":    index,
			"result":   result,
			"type":     strings.Join(remainingMods, "") + templates.CurrentImports.LookupType(t.Definition.GoType),
			"next":     t.unmarshal(result+"["+index+"]", rawIf+"["+index+"]", remainingMods[1:], depth+1),
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
	return t.middleware(result, raw, t.Modifiers, 1)
}

func (t TypeReference) middleware(result, raw string, remainingMods []string, depth int) string {
	if len(remainingMods) == 1 && remainingMods[0] == modPtr {
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
	switch {
	case len(remainingMods) > 0 && remainingMods[0] == modPtr:
		return tpl(`if {{.raw}} != nil {
				{{.next}}
			}`, map[string]interface{}{
			"t":      t,
			"raw":    raw,
			"result": result,
			"mods":   strings.Join(remainingMods[1:], ""),
			"next":   t.middleware(result, raw, remainingMods[1:], depth+1),
		})

	case len(remainingMods) > 0 && remainingMods[0] == modList:
		var index = "idx" + strconv.Itoa(depth)

		return tpl(`for {{.index}} := range {{.raw}} {
				{{ .next -}}
			}`, map[string]interface{}{
			"raw":    raw,
			"index":  index,
			"result": result,
			"type":   strings.Join(remainingMods, "") + templates.CurrentImports.LookupType(t.Definition.GoType),
			"next":   t.middleware(result+"["+index+"]", raw+"["+index+"]", remainingMods[1:], depth+1),
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
