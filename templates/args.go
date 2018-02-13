package templates

var argsTpl = `
{{- define "args" }}
	{{- range $i, $arg := . }}
		var arg{{$i}} {{$arg.Signature }}
		{{- if eq $arg.FullName "time.Time" }}
			if tmp, ok := field.Args[{{$arg.GQLName|quote}}]; ok {
				if tmpStr, ok := tmp.(string); ok {
					tmpDate, err := time.Parse(time.RFC3339, tmpStr)
					if err != nil {
						ec.Error(err)
						continue
					}
					arg{{$i}} = {{if $arg.Type.IsPtr}}&{{end}}tmpDate
				} else {
					ec.Errorf("Time '{{$arg.GQLName}}' should be RFC3339 formatted string")
					continue
				}
			}
		{{- else if eq $arg.GoType "map[string]interface{}" }}
			if tmp, ok := field.Args[{{$arg.GQLName|quote}}]; ok {
				{{- if $arg.Type.IsPtr }}
					tmp2 := tmp.({{$arg.GoType}})
					arg{{$i}} = &tmp2
				{{- else }}
					arg{{$i}} = tmp.({{$arg.GoType}})
				{{- end }}
			}
		{{- else if $arg.IsScalar }}
			if tmp, ok := field.Args[{{$arg.GQLName|quote}}]; ok {
				tmp2, err := coerce{{$arg.GoType|ucFirst}}(tmp)
				if err != nil {
					ec.Error(err)
					continue
				}
				arg{{$i}} = {{if $arg.Type.IsPtr}}&{{end}}tmp2
			}
		{{- else }}
			err := unpackComplexArg(&arg{{$i}}, field.Args[{{$arg.GQLName|quote}}])
			if err != nil {
				ec.Error(err)
				continue
			}
		{{- end}}
	{{- end }}
{{- end }}
`
