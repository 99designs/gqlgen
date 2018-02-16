package templates

const inputTpl = `
{{- define "input" }}
	{{- if .IsMarshaled }} 
	func Unmarshal{{ .GQLType }}(v interface{}) ({{.FullName}}, error) {
		var it {{.FullName}}
	
		for k, v := range v.(map[string]interface{}) {
			switch k {
			{{- range $field := .Fields }}
			case {{$field.GQLName|quote}}:
				{{$field.Unmarshal "val" "v" }}
				if err != nil {
					return it, err
				}
				{{$field.GoVarName}} = {{if $field.Type.IsPtr}}&{{end}}val
			{{- end }}
			}
		} 
	
		return it, nil
	}
	{{- end }}
{{- end }}
`
