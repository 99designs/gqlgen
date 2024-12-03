//go:generate go run ../../../testdata/gqlgen.go -config gqlgen_default.yml -stub generated_default/stub.go
//go:generate go run ../../../testdata/gqlgen.go -config gqlgen_compliant.yml -stub generated_compliant/stub.go
//go:generate go run ../../../testdata/gqlgen.go -config gqlgen_compliant_input_int.yml -stub generated_compliant_input_int/stub.go

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
				"EchoIntToInt":                "func(ctx context.Context, n *int) (int, error)",
				"EchoInt64ToInt64":            "func(ctx context.Context, n *int) (int, error)",
				"EchoIntInputToIntObject":     "func(ctx context.Context, input Input) (*Result, error)",
				"EchoInt64InputToInt64Object": "func(ctx context.Context, input Input64) (*Result64, error)",
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
				"EchoIntToInt":                "func(ctx context.Context, n *int32) (int32, error)",
				"EchoInt64ToInt64":            "func(ctx context.Context, n *int) (int, error)",
				"EchoIntInputToIntObject":     "func(ctx context.Context, input Input) (*Result, error)",
				"EchoInt64InputToInt64Object": "func(ctx context.Context, input Input64) (*Result64, error)",
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
				"EchoIntToInt":                "func(ctx context.Context, n *int) (int32, error)",
				"EchoInt64ToInt64":            "func(ctx context.Context, n *int) (int, error)",
				"EchoIntInputToIntObject":     "func(ctx context.Context, input Input) (*Result, error)",
				"EchoInt64InputToInt64Object": "func(ctx context.Context, input Input64) (*Result64, error)",
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
			resolver, models := parsePackage(t, tc.pkgPath)

			t.Run("resolver signature", func(t *testing.T) {
				fieldMap := make(map[string]string)
				for _, field := range resolver.Fields.List {
					fn, ok := field.Type.(*ast.FuncType)
					require.True(t, ok)

					sig, err := printNode(fn)
					require.NoError(t, err)

					fieldMap[join(field.Names)] = sig
				}

				require.Equal(t, tc.signature, fieldMap)
			})

			t.Run("models", func(t *testing.T) {
				fieldMap := make(map[string][]string)
				for name, model := range models {
					var fields []string
					for _, field := range model.Fields.List {
						typ, err := printNode(field.Type)
						require.NoError(t, err)
						fields = append(fields, join(field.Names)+" "+typ)
					}
					fieldMap[name] = fields
				}

				require.Equal(t, tc.models, fieldMap)
			})
		})
	}
}

func parsePackage(t *testing.T, pkgPath string) (stubResolver *ast.StructType, models map[string]*ast.StructType) {
	t.Helper()

	path, err := filepath.Abs(pkgPath)
	require.NoError(t, err)

	pkgs, err := parser.ParseDir(token.NewFileSet(), path, nil, parser.AllErrors)
	require.NoError(t, err)

	pkg, ok := pkgs["generated"]
	require.True(t, ok, fmt.Sprintf("invalid package found at %v", pkgPath))

	var stub *ast.StructType
	models = make(map[string]*ast.StructType)
	ast.Inspect(pkg, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.TypeSpec:
			s, ok := node.Type.(*ast.StructType)
			if !ok {
				return true
			}
			if node.Name.Name == "Stub" {
				stub = s
			} else if slices.Contains(
				[]string{"Input", "Input64", "Result", "Result64"},
				node.Name.Name,
			) {
				models[node.Name.Name] = s
			}
			return true
		default:
		}
		return true
	})
	require.NotNil(t, stub, fmt.Sprintf("could not find stub object in %v", pkg))

	var resolverField *ast.Field
	for _, field := range stub.Fields.List {
		if join(field.Names) == "QueryResolver" {
			resolverField = field
			break
		}
	}
	require.NotNil(t, resolverField, fmt.Sprintf("could not find QueryResolver field in Stub object in %v", pkg))

	stubResolver, ok = resolverField.Type.(*ast.StructType)
	require.True(t, ok, fmt.Sprintf("QueryResolver field in Stub object in %v is not a struct", pkg))

	return stubResolver, models
}

func printNode(node interface{}) (string, error) {
	buf := &bytes.Buffer{}
	err := format.Node(buf, token.NewFileSet(), node)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func join[T fmt.Stringer](s []T) string {
	var sb strings.Builder
	for _, v := range s {
		sb.WriteString(v.String())
	}
	return sb.String()
}
