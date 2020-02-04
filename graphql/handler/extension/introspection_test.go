package extension

import (
	"context"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql"

	"github.com/99designs/gqlgen/graphql/introspection"
	"github.com/vektah/gqlparser/ast"

	"github.com/stretchr/testify/require"
)

func TestIntrospection(t *testing.T) {
	rc := &graphql.OperationContext{
		DisableIntrospection: true,
	}
	require.Nil(t, Introspection{}.MutateOperationContext(context.Background(), rc))
	require.Equal(t, false, rc.DisableIntrospection)
}

func TestIntrospection_InterceptField(t *testing.T) {
	type fields struct {
		AllowFieldFunc      func(ctx context.Context, t *introspection.Type, field *introspection.Field) (bool, error)
		AllowInputValueFunc func(ctx context.Context, t *introspection.Type, inputValue *introspection.InputValue) (bool, error)
	}
	type args struct {
		kind ast.DefinitionKind
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantRes []string
	}{
		{
			name: "field",
			fields: fields{
				AllowFieldFunc: func(ctx context.Context, t *introspection.Type, field *introspection.Field) (b bool, err error) {
					if *t.Name() == "TestType1" {
						return !strings.HasSuffix(field.Name, "1"), nil
					}
					return true, nil
				},
			},
			args:    args{kind: ast.Object},
			wantRes: []string{"testField2", "testField3"},
		},
		{
			name: "inputValue",
			fields: fields{
				AllowInputValueFunc: func(ctx context.Context, t *introspection.Type, inputValue *introspection.InputValue) (b bool, err error) {
					if *t.Name() == "TestType1" {
						return !strings.HasSuffix(inputValue.Name, "1"), nil
					}
					return true, nil
				},
			},
			args:    args{kind: ast.InputObject},
			wantRes: []string{"testField2", "testField3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Introspection{
				AllowFieldFunc:      tt.fields.AllowFieldFunc,
				AllowInputValueFunc: tt.fields.AllowInputValueFunc,
			}
			typ := introspection.WrapTypeFromDef(nil, &ast.Definition{
				Kind: tt.args.kind,
				Name: "TestType1",
				Fields: []*ast.FieldDefinition{
					{Name: "testField1"},
					{Name: "testField2"},
					{Name: "testField3"},
				},
			})
			ctx := graphql.WithFieldContext(context.Background(), &graphql.FieldContext{Result: typ})
			ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{
				Field: graphql.CollectedField{
					Field: &ast.Field{
						Name: tt.name,
					},
				},
			})
			gotRes, err := c.InterceptField(ctx, func(ctx context.Context) (res interface{}, err error) {
				switch tt.args.kind {
				case ast.Object:
					return typ.Fields(false), nil
				case ast.InputObject:
					return typ.InputFields(), nil
				}
				require.Fail(t, "unexpected ast.DefinitionKind: %v", tt.args.kind)
				return nil, nil
			})
			require.NoError(t, err)

			var actualFields []string
			switch tt.args.kind {
			case ast.Object:
				for _, field := range gotRes.([]introspection.Field) {
					actualFields = append(actualFields, field.Name)
				}
			case ast.InputObject:
				for _, field := range gotRes.([]introspection.InputValue) {
					actualFields = append(actualFields, field.Name)
				}
			default:
				require.FailNow(t, "", "unexpected ast.DefinitionKind: %v", tt.args.kind)
			}
			require.Equal(t, tt.wantRes, actualFields)
		})
	}
}
