package templates

const objectTpl = `
{{- define "object" }}
{{ $object := . }}

var {{ $object.Type.GraphQLName|lcFirst}}Implementors = {{$object.Implementors}}

// nolint: gocyclo, errcheck, gas, goconst
func (ec *executionContext) _{{$object.Type.GraphQLName|lcFirst}}(sel []query.Selection, it *{{$object.Type.Local}}) jsonw.Writer {
	fields := ec.collectFields(sel, {{$object.Type.GraphQLName|lcFirst}}Implementors, map[string]bool{})
	out := jsonw.NewOrderedMap(len(fields))
	for i, field := range fields {
		out.Keys[i] = field.Alias
		out.Values[i] = jsonw.Null

		switch field.Name {
		{{- range $field := $object.Fields }}
		case "{{$field.GraphQLName}}":
			{{- template "args" $field.Args }}

			{{- if $field.IsConcurrent }}
				ec.wg.Add(1)
				go func(i int, field collectedField) {
					defer ec.wg.Done()
			{{- end }}

			{{- if $field.VarName }}
				res := {{$field.VarName}}
			{{- else if $field.MethodName }}
				{{- if $field.NoErr }}
					res := {{$field.MethodName}}({{ $field.CallArgs }})
				{{- else }}
					res, err := {{$field.MethodName}}({{ $field.CallArgs }})
					if err != nil {
						ec.Error(err)
						{{ if $field.IsConcurrent }}return{{ else }}continue{{end}}
					}
				{{- end }}
			{{- else }}
				res, err := ec.resolvers.{{ $object.Name }}_{{ $field.GraphQLName }}({{ $field.CallArgs }})
				if err != nil {
					ec.Error(err)
					{{ if $field.IsConcurrent }}return{{ else }}continue{{end}}
				}
			{{- end }}

			{{ $field.WriteJson "out.Values[i]" }}

			{{- if $field.IsConcurrent }}
				}(i, field)
			{{- end }}
		{{- end }}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}

	return out
}


{{- end}}
`
