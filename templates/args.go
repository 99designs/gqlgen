package templates

const argsTpl = `
{{- define "args" }}
	{{- range $i, $arg := . }}
		var arg{{$i}} {{$arg.Signature }}
		{{- if eq $arg.GoType "map[string]interface{}" }}
			if tmp, ok := field.Args[{{$arg.GQLName|quote}}]; ok {
				{{- if $arg.Type.IsPtr }}
					tmp2 := tmp.({{$arg.GoType}})
					arg{{$i}} = &tmp2
				{{- else }}
					arg{{$i}} = tmp.({{$arg.GoType}})
				{{- end }}
			}
		{{- else}}
			if tmp, ok := field.Args[{{$arg.GQLName|quote}}]; ok {
				{{$arg.Unmarshal "tmp2" "tmp" }}
				if err != nil {
					badArgs = true
				}
				arg{{$i}} = {{if $arg.Type.IsPtr}}&{{end}}tmp2
			}
		{{- end}}
	{{- end }}
{{- end }}
`
