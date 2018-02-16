package templates

const objectTpl = `
{{- define "object" }}
{{ $object := . }}

var {{ $object.GQLType|lcFirst}}Implementors = {{$object.Implementors}}

// nolint: gocyclo, errcheck, gas, goconst
func (ec *executionContext) _{{$object.GQLType|lcFirst}}(sel []query.Selection, it *{{$object.FullName}}) jsonw.Writer {
	fields := ec.collectFields(sel, {{$object.GQLType|lcFirst}}Implementors, map[string]bool{})
	out := jsonw.NewOrderedMap(len(fields))
	for i, field := range fields {
		out.Keys[i] = field.Alias
		out.Values[i] = jsonw.Null

		switch field.Name {
		case "__typename":
			out.Values[i] = jsonw.String({{$object.GQLType|quote}})
		{{- range $field := $object.Fields }}
		case "{{$field.GQLName}}":
			{{- template "args" $field.Args }}

			{{- if $field.IsConcurrent }}
				ec.wg.Add(1)
				go func(i int, field collectedField) {
					defer ec.wg.Done()
			{{- end }}

			{{- if $field.GoVarName }}
				res := {{$field.GoVarName}}
			{{- else if $field.GoMethodName }}
				{{- if $field.NoErr }}
					res := {{$field.GoMethodName}}({{ $field.CallArgs }})
				{{- else }}
					res, err := {{$field.GoMethodName}}({{ $field.CallArgs }})
					if err != nil {
						ec.Error(err)
						{{ if $field.IsConcurrent }}return{{ else }}continue{{end}}
					}
				{{- end }}
			{{- else }}
				res, err := ec.resolvers.{{ $object.GQLType }}_{{ $field.GQLName }}({{ $field.CallArgs }})
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
