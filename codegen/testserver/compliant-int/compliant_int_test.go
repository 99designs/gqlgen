//go:generate go run ../../../testdata/gqlgen.go -config gqlgen_default.yml
//go:generate go run ../../../testdata/gqlgen.go -config gqlgen_compliant.yml
//go:generate go run ../../../testdata/gqlgen.go -config gqlgen_compliant_input_int.yml

package compliant_int

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCodegen(t *testing.T) {
	cases := []struct {
		name      string
		pkgPath   string
		signature map[string]string
		models    map[string][]string
	}{
		{
			name:    "no model configuration default generation",
			pkgPath: "generated_default",
			signature: map[string]string{
				"EchoIntToInt":     "func(ctx context.Context, n *int) (int, error)",
				"EchoInt64ToInt64": "func(ctx context.Context, n *int) (int, error)",
			},
			models: map[string][]string{
				"Input":    {"N *int"},
				"Result":   {"N int"},
				"Input64":  {"N *int"},
				"Result64": {"N int"},
			},
		},
		{
			name:    "compliant model configuration in yaml",
			pkgPath: "generated_compliant",
			signature: map[string]string{
				"EchoIntToInt":     "func(ctx context.Context, n *int32) (int32, error)",
				"EchoInt64ToInt64": "func(ctx context.Context, n *int) (int, error)",
			},
			models: map[string][]string{
				"Input":    {"N *int32"},
				"Result":   {"N int32"},
				"Input64":  {"N *int"},
				"Result64": {"N int"},
			},
		},
		{
			name:    "compliant model configuration with int input setting",
			pkgPath: "generated_compliant_input_int",
			signature: map[string]string{
				"EchoIntToInt":     "func(ctx context.Context, n *int) (int32, error)",
				"EchoInt64ToInt64": "func(ctx context.Context, n *int) (int, error)",
			},
			models: map[string][]string{
				"Input":    {"N *int"},
				"Result":   {"N int32"},
				"Input64":  {"N *int"},
				"Result64": {"N int"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path, err := filepath.Abs(tc.pkgPath)
			require.NoError(t, err)

			pkgs, err := parser.ParseDir(token.NewFileSet(), path, nil, parser.AllErrors)
			require.NoError(t, err)

			pkg, ok := pkgs["generated"]
			require.True(t, ok, fmt.Sprintf("invalid package found at %v", tc.pkgPath))

			modelsMap := make(map[string][]string)
			signatureMap := make(map[string]string)
			ast.Inspect(pkg, func(node ast.Node) bool {
				switch node := node.(type) {
				case *ast.FuncDecl:
					if slices.Contains(
						[]string{"EchoIntToInt", "EchoInt64ToInt64"},
						node.Name.Name,
					) {
						signatureMap[node.Name.Name] = printNode(t, node.Type)
					}
				case *ast.TypeSpec:
					s, ok := node.Type.(*ast.StructType)
					if !ok {
						return true
					}
					if slices.Contains(
						[]string{"Input", "Input64", "Result", "Result64"},
						node.Name.Name,
					) {
						var fields []string
						for _, field := range s.Fields.List {
							fields = append(fields, join(field.Names)+" "+printNode(t, field.Type))
						}
						modelsMap[node.Name.Name] = fields
					}
					return true
				default:
				}
				return true
			})

			t.Run("resolver signature", func(t *testing.T) {
				require.Equal(t, tc.signature, signatureMap)
			})
			t.Run("models", func(t *testing.T) {
				require.Equal(t, tc.models, modelsMap)
			})
		})
	}
}

func printNode(t *testing.T, node interface{}) string {
	t.Helper()

	buf := &bytes.Buffer{}
	err := format.Node(buf, token.NewFileSet(), node)
	require.NoError(t, err)

	return buf.String()
}

func join[T fmt.Stringer](s []T) string {
	var sb strings.Builder
	for _, v := range s {
		sb.WriteString(v.String())
	}
	return sb.String()
}
