package templates

const modelTpl = `
{{- define "model" }}
	type {{.GoType}} struct {
		{{- range $field := .Fields }}
			{{- if $field.GoVarName }} 
				{{ $field.GoVarName }} {{$field.Signature}}
			{{- else }} 
				{{ $field.GoFKName }} {{$field.GoFKType}}
			{{- end }}
		{{- end }}
	}
{{- end }}
`
