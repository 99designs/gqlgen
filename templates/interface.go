package templates

const interfaceTpl = `
{{- define "interface"}}
{{- $interface := . }}

func (ec *executionContext) _{{$interface.GQLType|lcFirst}}(sel []query.Selection, it *{{$interface.FullName}}) graphql.Marshaler {
	switch it := (*it).(type) {
	case nil:
		return graphql.Null
	{{- range $implementor := $interface.Implementors }}
	case {{$implementor.FullName}}:
		return ec._{{$implementor.GQLType|lcFirst}}(sel, &it)

	case *{{$implementor.FullName}}:
		return ec._{{$implementor.GQLType|lcFirst}}(sel, it)

	{{- end }}
	default:
		panic(fmt.Errorf("unexpected type %T", it))
	}
}

{{- end}}
`
