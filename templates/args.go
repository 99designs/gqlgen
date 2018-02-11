package templates

var argsTpl = `
{{- define "args" }}
	{{- range $i, $arg := . }}
		var arg{{$i}} {{$arg.Type.Local }}
		{{- if eq $arg.Type.FullName "time.Time" }}
			if tmp, ok := field.Args[{{$arg.Name|quote}}]; ok {
				if tmpStr, ok := tmp.(string); ok {
					tmpDate, err := time.Parse(time.RFC3339, tmpStr)
					if err != nil {
						ec.Error(err)
						continue
					}
					arg{{$i}} = {{if $arg.Type.IsPtr}}&{{end}}tmpDate
				} else {
					ec.Errorf("Time '{{$arg.Name}}' should be RFC3339 formatted string")
					continue
				}
			}
		{{- else if eq $arg.Type.Name "map[string]interface{}" }}
			if tmp, ok := field.Args[{{$arg.Name|quote}}]; ok {
				{{- if $arg.Type.IsPtr }}
					tmp2 := tmp.({{$arg.Type.Name}})
					arg{{$i}} = &tmp2
				{{- else }}
					arg{{$i}} = tmp.({{$arg.Type.Name}})
				{{- end }}
			}
		{{- else if $arg.Type.Scalar }}
			if tmp, ok := field.Args[{{$arg.Name|quote}}]; ok {
				tmp2, err := coerce{{$arg.Type.Name|ucFirst}}(tmp)
				if err != nil {
					ec.Error(err)
					continue
				}
				arg{{$i}} = {{if $arg.Type.IsPtr}}&{{end}}tmp2
			}
		{{- else }}
			err := unpackComplexArg(&arg{{$i}}, field.Args[{{$arg.Name|quote}}])
			if err != nil {
				ec.Error(err)
				continue
			}
		{{- end}}
	{{- end }}
{{- end }}
`
