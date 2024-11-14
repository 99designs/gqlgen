package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecConfig_Check(t *testing.T) {
	type fields struct {
		Package          string
		Layout           ExecLayout
		ExecTemplate     string
		ExecTemplateDir  string
		Filename         string
		FilenameTemplate string
		DirName          string
	}
	tests := []struct {
		name   string
		fields fields
		err    error
	}{
		{
			name: "single-file layout without filename",
			fields: fields{
				Layout: ExecLayoutSingleFile,
			},
			err: errors.New("filename must be specified when using single-file layout"),
		},
		{
			name: "single-file layout with invalid filename",
			fields: fields{
				Layout:   ExecLayoutSingleFile,
				Filename: "output.txt",
			},
			err: errors.New("filename should be path to a go source file when using single-file layout"),
		},
		{
			name: "follow-schema layout without dir",
			fields: fields{
				Layout: ExecLayoutFollowSchema,
			},
			err: errors.New("dir must be specified when using follow-schema layout"),
		},
		{
			name: "invalid layout",
			fields: fields{
				Layout: "invalid-layout",
			},
			err: errors.New("invalid layout invalid-layout"),
		},
		{
			name: "package with invalid characters",
			fields: fields{
				Filename: "generated.go",
				Package:  "invalid/package",
			},
			err: errors.New("package should be the output package name only, do not include the output filename"),
		},
		{
			name: "exec_template and exec_template_dir both defined",
			fields: fields{
				Filename:        "generated.go",
				ExecTemplate:    "template",
				ExecTemplateDir: "template_dir",
			},
			err: errors.New("exec_template and exec_template_dir cannot be defined at the same time"),
		},
		{
			name: "valid single-file layout",
			fields: fields{
				Layout:   ExecLayoutSingleFile,
				Filename: "generated.go",
			},
			err: nil,
		},
		{
			name: "valid follow-schema layout",
			fields: fields{
				Layout:  ExecLayoutFollowSchema,
				DirName: "output_dir",
			},
			err: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ExecConfig{
				Package:          tt.fields.Package,
				Layout:           tt.fields.Layout,
				ExecTemplate:     tt.fields.ExecTemplate,
				ExecTemplateDir:  tt.fields.ExecTemplateDir,
				Filename:         tt.fields.Filename,
				FilenameTemplate: tt.fields.FilenameTemplate,
				DirName:          tt.fields.DirName,
			}
			err := r.Check()
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
