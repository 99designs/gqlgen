package rewrite

import (
	"go/ast"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

func TestRewriter(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		r, err := New("testdata")
		require.NoError(t, err)

		body := r.GetMethodBody("Foo", "Method")
		require.Equal(t, `
	// leading comment

	// field comment
	m.Field++

	// trailing comment
`, strings.ReplaceAll(body, "\r\n", "\n"))

		imps := r.ExistingImports("testdata/example.go")
		require.Len(t, imps, 2)
		assert.Equal(t, []Import{
			{
				Alias:      "lol",
				ImportPath: "bytes",
			},
			{
				Alias:      "",
				ImportPath: "fmt",
			},
		}, imps)
	})

	t.Run("out of scope dir", func(t *testing.T) {
		_, err := New("../../../out-of-gomod/package")
		require.Error(t, err)
	})
}

func TestRewriter_GetMethodComment(t *testing.T) {
	type fields struct {
		pkg    *packages.Package
		files  map[string]string
		copied map[ast.Decl]bool
	}
	type args struct {
		structname string
		methodname string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "comments",
			fields: fields{
				pkg: &packages.Package{
					Syntax: []*ast.File{
						{
							Decls: []ast.Decl{
								&ast.FuncDecl{
									Name: &ast.Ident{Name: "Method"},
									Doc: &ast.CommentGroup{
										List: []*ast.Comment{
											{
												Text: "// comment",
											},
											{
												Text: "// comment",
											},
										},
									},
									Recv: &ast.FieldList{
										List: []*ast.Field{
											{
												Type: &ast.Ident{Name: "Foo"},
											},
										},
									},
								},
							},
						},
					},
				},
				copied: map[ast.Decl]bool{},
			},
			args: args{
				structname: "Foo",
				methodname: "Method",
			},
			want: " comment\n comment", //nolint: dupword // this is for the test
		},
		{
			name: "directive in comment",
			fields: fields{
				pkg: &packages.Package{
					Syntax: []*ast.File{
						{
							Decls: []ast.Decl{
								&ast.FuncDecl{
									Name: &ast.Ident{Name: "Method"},
									Doc: &ast.CommentGroup{
										List: []*ast.Comment{
											{
												Text: "// comment",
											},
											{
												Text: "//nolint:test // test",
											},
										},
									},
									Recv: &ast.FieldList{
										List: []*ast.Field{
											{
												Type: &ast.Ident{Name: "Foo"},
											},
										},
									},
								},
							},
						},
					},
				},
				copied: map[ast.Decl]bool{},
			},
			args: args{
				structname: "Foo",
				methodname: "Method",
			},
			want: " comment\nnolint:test // test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Rewriter{
				pkg:    tt.fields.pkg,
				files:  tt.fields.files,
				copied: tt.fields.copied,
			}
			assert.Equalf(t, tt.want, r.GetMethodComment(tt.args.structname, tt.args.methodname), "GetMethodComment(%v, %v)", tt.args.structname, tt.args.methodname)
		})
	}
}
