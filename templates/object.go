package templates

const objectTpl = `
{{- define "object" }}
{{ $object := . }}

var {{ $object.GQLType|lcFirst}}Implementors = {{$object.Implementors}}

// nolint: gocyclo, errcheck, gas, goconst
{{- if .Stream }}
func (ec *executionContext) _{{$object.GQLType|lcFirst}}(sel []query.Selection, it *{{$object.FullName}}) <-chan graphql.Marshaler {
	fields := graphql.CollectFields(ec.doc, sel, {{$object.GQLType|lcFirst}}Implementors, ec.variables)

	if len(fields) != 1 {
		ec.Errorf("must subscribe to exactly one stream")
		return nil
	}

	var field = fields[0]
	channel := make(chan graphql.Marshaler, 1)
	switch field.Name {
	{{- range $field := $object.Fields }}
	case "{{$field.GQLName}}":
		badArgs := false
		{{- template "args" $field.Args }}
		if badArgs {
			return nil
		}

		{{- if $field.GoVarName }}
			results := {{$field.GoVarName}}
		{{- else if $field.GoMethodName }}
			{{- if $field.NoErr }}
				results := {{$field.GoMethodName}}({{ $field.CallArgs }})
			{{- else }}
				results, err := {{$field.GoMethodName}}({{ $field.CallArgs }})
				if err != nil {
					ec.Error(err)
					return nil
				}
			{{- end }}
		{{- else }}
			results, err := ec.resolvers.{{ $object.GQLType }}_{{ $field.GQLName }}({{ $field.CallArgs }})
			if err != nil {
				ec.Error(err)
				return nil
			}
		{{- end }}

		go func() {
			for res := range results {
				var out graphql.OrderedMap
				var messageRes graphql.Marshaler
				{{ $field.WriteJson "messageRes" }}
				out.Add(field.Alias, messageRes)
				channel <- &out
			}
		}()

	{{- end }}
	default:
		panic("unknown field " + strconv.Quote(field.Name))
	}

	return channel
}
{{- else }}
func (ec *executionContext) _{{$object.GQLType|lcFirst}}(sel []query.Selection, it *{{$object.FullName}}) graphql.Marshaler {
	fields := graphql.CollectFields(ec.doc, sel, {{$object.GQLType|lcFirst}}Implementors, ec.variables)
	out := graphql.NewOrderedMap(len(fields))
	for i, field := range fields {
		out.Keys[i] = field.Alias
		out.Values[i] = graphql.Null

		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString({{$object.GQLType|quote}})
		{{- range $field := $object.Fields }}
		case "{{$field.GQLName}}":
			badArgs := false
			{{- template "args" $field.Args }}
			if badArgs {
				continue
			}

			{{- if $field.IsConcurrent }}
				ec.wg.Add(1)
				go func(i int, field graphql.CollectedField) {
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
{{- end }}

{{- end}}
`
