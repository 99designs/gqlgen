package templates

const interfaceTpl = `
{{- define "interface"}}
{{- $interface := . }}

func (ec *executionContext) _{{$interface.Type.GraphQLName|lcFirst}}(sel []query.Selection, it *{{$interface.Type.Local}}) jsonw.Writer {
	switch it := (*it).(type) {
	case nil:
		return jsonw.Null
	{{- range $implementor := $interface.Type.Implementors }}
	case {{$implementor.Local}}:
		return ec._{{$implementor.GraphQLName|lcFirst}}(sel, &it)

	case *{{$implementor.Local}}:
		return ec._{{$implementor.GraphQLName|lcFirst}}(sel, it)

	{{- end }}
	default:
		panic(fmt.Errorf("unexpected type %T", it))
	}
}

{{- end}}
`
