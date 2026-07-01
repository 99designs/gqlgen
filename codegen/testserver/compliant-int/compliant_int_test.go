//go:generate go run ../../../testdata/gqlgen.go -config gqlgen_default.yml -stub generated-default/stub.go
//go:generate go run ../../../testdata/gqlgen.go -config gqlgen_compliant_strict.yml -stub generated-compliant-strict/stub.go

package compliant_int

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"

	"github.com/99designs/gqlgen/client"
	gencompliant "github.com/99designs/gqlgen/codegen/testserver/compliant-int/generated-compliant-strict"
	gendefault "github.com/99designs/gqlgen/codegen/testserver/compliant-int/generated-default"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

// genImportBase is the import-path prefix of the generated fixture packages
// inspected by TestCodegen. Their directory names (generated-default,
// generated-compliant-strict) differ from their package name (generated).
const genImportBase = "github.com/99designs/gqlgen/codegen/testserver/compliant-int/"

func TestCodegen(t *testing.T) {
	cases := []struct {
		name      string
		pkgPath   string
		signature map[string]string
		models    map[string][]string
	}{
		{
			name:    "no model configuration default generation",
			pkgPath: "generated-default",
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
			name:    "strict compliant model configuration in yaml",
			pkgPath: "generated-compliant-strict",
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
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &packages.Config{
				Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax,
			}
			loaded, err := packages.Load(cfg, genImportBase+tc.pkgPath)
			require.NoError(t, err)
			require.Len(t, loaded, 1)

			pkg := loaded[0]
			require.Empty(t, pkg.Errors)
			require.Equal(t, "generated", pkg.Name, "unexpected package at %v", tc.pkgPath)

			modelsMap := make(map[string][]string)
			signatureMap := make(map[string]string)
			inspect := func(node ast.Node) bool {
				switch node := node.(type) {
				case *ast.FuncDecl:
					if slices.Contains(
						[]string{"EchoIntToInt", "EchoInt64ToInt64"},
						node.Name.Name,
					) {
						signatureMap[node.Name.Name] = printNode(t, pkg.Fset, node.Type)
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
							fields = append(
								fields,
								join(field.Names)+" "+printNode(t, pkg.Fset, field.Type),
							)
						}
						modelsMap[node.Name.Name] = fields
					}
					return true
				default:
				}
				return true
			}
			for _, file := range pkg.Syntax {
				ast.Inspect(file, inspect)
			}

			t.Run("resolver signature", func(t *testing.T) {
				require.Equal(t, tc.signature, signatureMap)
			})
			t.Run("models", func(t *testing.T) {
				require.Equal(t, tc.models, modelsMap)
			})
		})
	}
}

func TestIntegration(t *testing.T) {
	defaultStub := &gendefault.Stub{}
	defaultStub.QueryResolver.EchoIntToInt = func(_ context.Context, n *int) (int, error) {
		if n == nil {
			return 0, nil
		}
		return *n, nil
	}
	strictCompliantStub := &gencompliant.Stub{}
	strictCompliantStub.QueryResolver.EchoIntToInt = func(_ context.Context, n *int32) (int32, error) {
		if n == nil {
			return 0, nil
		}
		return *n, nil
	}

	cases := []struct {
		name      string
		exec      graphql.ExecutableSchema
		willError bool
	}{
		{
			name:      "default generation allows int32 overflow inputs",
			exec:      gendefault.NewExecutableSchema(gendefault.Config{Resolvers: defaultStub}),
			willError: false,
		},
		{
			name: "strict compliant generation does not allow int32 overflow inputs",
			exec: gencompliant.NewExecutableSchema(
				gencompliant.Config{Resolvers: strictCompliantStub},
			),
			willError: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := handler.New(tc.exec)
			srv.AddTransport(transport.POST{})

			c := client.New(srv)

			var resp struct {
				EchoIntToInt int
			}
			err := c.Post(`query { echoIntToInt(n: 2147483648) }`, &resp)
			if tc.willError {
				require.EqualError(
					t,
					err,
					`[{"message":"2147483648 overflows signed 32-bit integer","path":["echoIntToInt","n"]}]`,
				)
				require.Equal(t, 0, resp.EchoIntToInt)
				return
			}
			require.NoError(t, err)
			require.Equal(t, 2147483648, resp.EchoIntToInt)
		})
	}
}

func printNode(t *testing.T, fset *token.FileSet, node any) string {
	t.Helper()

	buf := &bytes.Buffer{}
	err := format.Node(buf, fset, node)
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
