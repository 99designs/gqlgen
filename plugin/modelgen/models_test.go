package modelgen

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/plugin/modelgen/internal/extrafields"
	"github.com/99designs/gqlgen/plugin/modelgen/out"
	"github.com/99designs/gqlgen/plugin/modelgen/out_enable_model_json_omitempty_tag_false"
	"github.com/99designs/gqlgen/plugin/modelgen/out_enable_model_json_omitempty_tag_nil"
	"github.com/99designs/gqlgen/plugin/modelgen/out_enable_model_json_omitempty_tag_true"
	"github.com/99designs/gqlgen/plugin/modelgen/out_nullable_input_omittable"
	"github.com/99designs/gqlgen/plugin/modelgen/out_struct_pointers"
)

func TestModelGeneration(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/gqlgen.yml")
	require.NoError(t, err)
	require.NoError(t, cfg.Init())
	p := Plugin{
		MutateHook: mutateHook,
		FieldHook:  DefaultFieldMutateHook,
	}
	require.NoError(t, p.MutateConfig(cfg))
	require.NoError(t, goBuild(t, "./out/"))

	require.True(t, cfg.Models.UserDefined("MissingTypeNotNull"))
	require.True(t, cfg.Models.UserDefined("MissingTypeNullable"))
	require.True(t, cfg.Models.UserDefined("MissingEnum"))
	require.True(t, cfg.Models.UserDefined("MissingUnion"))
	require.True(t, cfg.Models.UserDefined("MissingInterface"))
	require.True(t, cfg.Models.UserDefined("TypeWithDescription"))
	require.True(t, cfg.Models.UserDefined("EnumWithDescription"))
	require.True(t, cfg.Models.UserDefined("InterfaceWithDescription"))
	require.True(t, cfg.Models.UserDefined("UnionWithDescription"))
	require.True(t, cfg.Models.UserDefined("RenameFieldTest"))
	require.True(t, cfg.Models.UserDefined("ExtraFieldsTest"))

	t.Run("no pointer pointers", func(t *testing.T) {
		generated, err := os.ReadFile("./out/generated.go")
		require.NoError(t, err)
		require.NotContains(t, string(generated), "**")
	})

	t.Run("description is generated", func(t *testing.T) {
		node, err := parser.ParseFile(token.NewFileSet(), "./out/generated.go", nil, parser.ParseComments)
		require.NoError(t, err)
		for _, commentGroup := range node.Comments {
			text := commentGroup.Text()
			words := strings.Split(text, " ")
			require.Greaterf(t, len(words), 1, "expected description %q to have more than one word", text)
		}
	})

	t.Run("tags are applied", func(t *testing.T) {
		file, err := os.ReadFile("./out/generated.go")
		require.NoError(t, err)

		fileText := string(file)

		expectedTags := []string{
			`json:"missing2" database:"MissingTypeNotNullmissing2"`,
			`json:"name,omitempty" database:"MissingInputname"`,
			`json:"missing2,omitempty" database:"MissingTypeNullablemissing2"`,
			`json:"name,omitempty" database:"TypeWithDescriptionname"`,
		}

		for _, tag := range expectedTags {
			require.Contains(t, fileText, tag, "\nexpected:\n"+tag+"\ngot\n"+fileText)
		}
	})

	t.Run("field hooks are applied", func(t *testing.T) {
		file, err := os.ReadFile("./out/generated.go")
		require.NoError(t, err)

		fileText := string(file)

		expectedTags := []string{
			`json:"name,omitempty" anotherTag:"tag"`,
			`json:"enum,omitempty" yetAnotherTag:"12"`,
			`json:"noVal,omitempty" yaml:"noVal" repeated:"true"`,
			`json:"repeated,omitempty" someTag:"value" repeated:"true"`,
		}

		for _, tag := range expectedTags {
			require.Contains(t, fileText, tag, "\nexpected:\n"+tag+"\ngot\n"+fileText)
		}
	})

	t.Run("concrete types implement interface", func(t *testing.T) {
		var _ out.FooBarer = out.FooBarr{}
	})

	t.Run("implemented interfaces", func(t *testing.T) {
		pkg, err := parseAst("out")
		require.NoError(t, err)

		path := filepath.Join("out", "generated.go")
		generated := pkg.Files[path]

		type field struct {
			typ  string
			name string
		}
		cases := []struct {
			name       string
			wantFields []field
		}{
			{
				name: "A",
				wantFields: []field{
					{
						typ:  "method",
						name: "IsA",
					},
					{
						typ:  "method",
						name: "GetA",
					},
				},
			},
			{
				name: "B",
				wantFields: []field{
					{
						typ:  "method",
						name: "IsB",
					},
					{
						typ:  "method",
						name: "GetB",
					},
				},
			},
			{
				name: "C",
				wantFields: []field{
					{
						typ:  "method",
						name: "IsA",
					},
					{
						typ:  "method",
						name: "IsC",
					},
					{
						typ:  "method",
						name: "GetA",
					},
					{
						typ:  "method",
						name: "GetC",
					},
				},
			},
			{
				name: "D",
				wantFields: []field{
					{
						typ:  "method",
						name: "IsA",
					},
					{
						typ:  "method",
						name: "IsB",
					},
					{
						typ:  "method",
						name: "IsD",
					},
					{
						typ:  "method",
						name: "GetA",
					},
					{
						typ:  "method",
						name: "GetB",
					},
					{
						typ:  "method",
						name: "GetD",
					},
				},
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				typeSpec, ok := generated.Scope.Lookup(tc.name).Decl.(*ast.TypeSpec)
				require.True(t, ok)

				fields := typeSpec.Type.(*ast.InterfaceType).Methods.List
				for i, want := range tc.wantFields {
					if want.typ == "ident" {
						ident, ok := fields[i].Type.(*ast.Ident)
						require.True(t, ok)
						assert.Equal(t, want.name, ident.Name)
					}
					if want.typ == "method" {
						require.GreaterOrEqual(t, 1, len(fields[i].Names))
						name := fields[i].Names[0].Name
						assert.Equal(t, want.name, name)
					}
				}
			})
		}
	})

	t.Run("implemented interfaces type CDImplemented", func(t *testing.T) {
		pkg, err := parseAst("out")
		require.NoError(t, err)

		path := filepath.Join("out", "generated.go")
		generated := pkg.Files[path]

		wantMethods := []string{
			"IsA",
			"IsB",
			"IsC",
			"IsD",
		}

		gots := make([]string, 0, len(wantMethods))
		for _, decl := range generated.Decls {
			if funcDecl, ok := decl.(*ast.FuncDecl); ok {
				switch funcDecl.Name.Name {
				case "IsA", "IsB", "IsC", "IsD":
					gots = append(gots, funcDecl.Name.Name)
					require.Len(t, funcDecl.Recv.List, 1)
					recvIdent, ok := funcDecl.Recv.List[0].Type.(*ast.Ident)
					require.True(t, ok)
					require.Equal(t, "CDImplemented", recvIdent.Name)
				}
			}
		}

		sort.Strings(gots)
		require.Equal(t, wantMethods, gots)
	})

	t.Run("cyclical struct fields become pointers", func(t *testing.T) {
		require.Nil(t, out.CyclicalA{}.FieldOne)
		require.Nil(t, out.CyclicalA{}.FieldTwo)
		require.Nil(t, out.CyclicalA{}.FieldThree)
		require.NotNil(t, out.CyclicalA{}.FieldFour)
		require.Nil(t, out.CyclicalB{}.FieldOne)
		require.Nil(t, out.CyclicalB{}.FieldTwo)
		require.Nil(t, out.CyclicalB{}.FieldThree)
		require.Nil(t, out.CyclicalB{}.FieldFour)
		require.NotNil(t, out.CyclicalB{}.FieldFive)
	})

	t.Run("non-cyclical struct fields become pointers", func(t *testing.T) {
		require.NotNil(t, out.NotCyclicalB{}.FieldOne)
		require.Nil(t, out.NotCyclicalB{}.FieldTwo)
	})

	t.Run("recursive struct fields become pointers", func(t *testing.T) {
		require.Nil(t, out.Recursive{}.FieldOne)
		require.Nil(t, out.Recursive{}.FieldTwo)
		require.Nil(t, out.Recursive{}.FieldThree)
		require.NotNil(t, out.Recursive{}.FieldFour)
	})

	t.Run("overridden struct field names use same capitalization as config", func(t *testing.T) {
		require.NotNil(t, out.RenameFieldTest{}.GOODnaME)
	})

	t.Run("nullable input fields can be made omittable with goField", func(t *testing.T) {
		require.IsType(t, graphql.Omittable[*string]{}, out.MissingInput{}.NullString)
		require.IsType(t, graphql.Omittable[*out.MissingEnum]{}, out.MissingInput{}.NullEnum)
		require.IsType(t, graphql.Omittable[*out.ExistingInput]{}, out.MissingInput{}.NullObject)
	})

	t.Run("extra fields are present", func(t *testing.T) {
		var m out.ExtraFieldsTest

		require.IsType(t, int64(0), m.FieldInt)
		require.IsType(t, extrafields.Type{}, m.FieldInternalType)
		require.IsType(t, m.FieldStringPtr, new(string))
		require.IsType(t, []int64{}, m.FieldIntSlice)
	})
}

func TestModelGenerationOmitRootModels(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/gqlgen_omit_root_models.yml")
	require.NoError(t, err)
	require.NoError(t, cfg.Init())
	p := Plugin{
		MutateHook: mutateHook,
		FieldHook:  DefaultFieldMutateHook,
	}
	require.NoError(t, p.MutateConfig(cfg))
	require.NoError(t, goBuild(t, "./out/"))
	generated, err := os.ReadFile("./out/generated_omit_root_models.go")
	require.NoError(t, err)
	require.NotContains(t, string(generated), "type Mutation struct")
	require.NotContains(t, string(generated), "type Query struct")
	require.NotContains(t, string(generated), "type Subscription struct")
}

func TestModelGenerationOmitResolverFields(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/gqlgen_omit_resolver_fields.yml")
	require.NoError(t, err)
	require.NoError(t, cfg.Init())
	p := Plugin{
		MutateHook: mutateHook,
		FieldHook:  DefaultFieldMutateHook,
	}
	require.NoError(t, p.MutateConfig(cfg))
	require.NoError(t, goBuild(t, "./out_omit_resolver_fields/"))
	generated, err := os.ReadFile("./out_omit_resolver_fields/generated.go")
	require.NoError(t, err)
	require.Contains(t, string(generated), "type Base struct")
	require.Contains(t, string(generated), "StandardField")
	require.NotContains(t, string(generated), "ResolverField")
}

func TestModelGenerationStructFieldPointers(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/gqlgen_struct_field_pointers.yml")
	require.NoError(t, err)
	require.NoError(t, cfg.Init())
	p := Plugin{
		MutateHook: mutateHook,
		FieldHook:  DefaultFieldMutateHook,
	}
	require.NoError(t, p.MutateConfig(cfg))

	t.Run("no pointer pointers", func(t *testing.T) {
		generated, err := os.ReadFile("./out_struct_pointers/generated.go")
		require.NoError(t, err)
		require.NotContains(t, string(generated), "**")
	})

	t.Run("cyclical struct fields become pointers", func(t *testing.T) {
		require.Nil(t, out_struct_pointers.CyclicalA{}.FieldOne)
		require.Nil(t, out_struct_pointers.CyclicalA{}.FieldTwo)
		require.Nil(t, out_struct_pointers.CyclicalA{}.FieldThree)
		require.NotNil(t, out_struct_pointers.CyclicalA{}.FieldFour)
		require.Nil(t, out_struct_pointers.CyclicalB{}.FieldOne)
		require.Nil(t, out_struct_pointers.CyclicalB{}.FieldTwo)
		require.Nil(t, out_struct_pointers.CyclicalB{}.FieldThree)
		require.Nil(t, out_struct_pointers.CyclicalB{}.FieldFour)
		require.NotNil(t, out_struct_pointers.CyclicalB{}.FieldFive)
	})

	t.Run("non-cyclical struct fields do not become pointers", func(t *testing.T) {
		require.NotNil(t, out_struct_pointers.NotCyclicalB{}.FieldOne)
		require.NotNil(t, out_struct_pointers.NotCyclicalB{}.FieldTwo)
	})

	t.Run("recursive struct fields become pointers", func(t *testing.T) {
		require.Nil(t, out_struct_pointers.Recursive{}.FieldOne)
		require.Nil(t, out_struct_pointers.Recursive{}.FieldTwo)
		require.Nil(t, out_struct_pointers.Recursive{}.FieldThree)
		require.NotNil(t, out_struct_pointers.Recursive{}.FieldFour)
	})

	t.Run("no getters", func(t *testing.T) {
		generated, err := os.ReadFile("./out_struct_pointers/generated.go")
		require.NoError(t, err)
		require.NotContains(t, string(generated), "func (this")
	})
}

func TestModelGenerationNullableInputOmittable(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/gqlgen_nullable_input_omittable.yml")
	require.NoError(t, err)
	require.NoError(t, cfg.Init())
	p := Plugin{
		MutateHook: mutateHook,
		FieldHook:  DefaultFieldMutateHook,
	}
	require.NoError(t, p.MutateConfig(cfg))

	t.Run("nullable input fields are omittable", func(t *testing.T) {
		require.IsType(t, graphql.Omittable[*string]{}, out_nullable_input_omittable.MissingInput{}.Name)
		require.IsType(t, graphql.Omittable[*out_nullable_input_omittable.MissingEnum]{}, out_nullable_input_omittable.MissingInput{}.Enum)
		require.IsType(t, graphql.Omittable[*string]{}, out_nullable_input_omittable.MissingInput{}.NullString)
		require.IsType(t, graphql.Omittable[*out_nullable_input_omittable.MissingEnum]{}, out_nullable_input_omittable.MissingInput{}.NullEnum)
		require.IsType(t, graphql.Omittable[*out_nullable_input_omittable.ExistingInput]{}, out_nullable_input_omittable.MissingInput{}.NullObject)
	})

	t.Run("non-nullable input fields are not omittable", func(t *testing.T) {
		require.IsType(t, "", out_nullable_input_omittable.MissingInput{}.NonNullString)
	})
}

func TestModelGenerationOmitemptyConfig(t *testing.T) {
	suites := []struct {
		n       string
		cfg     string
		enabled bool
		t       any
	}{
		{
			n:       "nil",
			cfg:     "gqlgen_enable_model_json_omitempty_tag_nil.yml",
			enabled: true,
			t:       out_enable_model_json_omitempty_tag_nil.OmitEmptyJSONTagTest{},
		},
		{
			n:       "true",
			cfg:     "gqlgen_enable_model_json_omitempty_tag_true.yml",
			enabled: true,
			t:       out_enable_model_json_omitempty_tag_true.OmitEmptyJSONTagTest{},
		},
		{
			n:       "false",
			cfg:     "gqlgen_enable_model_json_omitempty_tag_false.yml",
			enabled: false,
			t:       out_enable_model_json_omitempty_tag_false.OmitEmptyJSONTagTest{},
		},
	}

	for _, s := range suites {
		t.Run(s.n, func(t *testing.T) {
			cfg, err := config.LoadConfig(fmt.Sprintf("testdata/%s", s.cfg))
			require.NoError(t, err)
			require.NoError(t, cfg.Init())
			p := Plugin{
				MutateHook: mutateHook,
				FieldHook:  DefaultFieldMutateHook,
			}
			require.NoError(t, p.MutateConfig(cfg))
			rt := reflect.TypeOf(s.t)

			// ensure non-nullable fields are never omitempty
			sfn, ok := rt.FieldByName("ValueNonNil")
			require.True(t, ok)
			require.Equal(t, "ValueNonNil", sfn.Tag.Get("json"))

			// test nullable fields for configured omitempty
			sf, ok := rt.FieldByName("Value")
			require.True(t, ok)

			var expected string
			if s.enabled {
				expected = "Value,omitempty"
			} else {
				expected = "Value"
			}
			require.Equal(t, expected, sf.Tag.Get("json"))
		})
	}
}

func mutateHook(b *ModelBuild) *ModelBuild {
	for _, model := range b.Models {
		for _, field := range model.Fields {
			field.Tag += ` database:"` + model.Name + field.Name + `"`
		}
	}

	return b
}

func parseAst(path string) (*ast.Package, error) {
	// test setup to parse the types
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, path, nil, parser.AllErrors)
	if err != nil {
		return nil, err
	}
	return pkgs["out"], nil
}

func goBuild(t *testing.T, path string) error {
	t.Helper()
	cmd := exec.Command("go", "build", path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(string(out))
	}

	return nil
}

func TestRemoveDuplicate(t *testing.T) {
	type args struct {
		t string
	}
	tests := []struct {
		name      string
		args      args
		want      string
		wantPanic bool
	}{
		{
			name: "Duplicate Test with 1",
			args: args{
				t: "json:\"name\"",
			},
			want: "json:\"name\"",
		},
		{
			name: "Duplicate Test with 2",
			args: args{
				t: "json:\"name\" json:\"name2\"",
			},
			want: "json:\"name2\"",
		},
		{
			name: "Duplicate Test with 3",
			args: args{
				t: "json:\"name\" json:\"name2\" json:\"name3\"",
			},
			want: "json:\"name3\"",
		},
		{
			name: "Duplicate Test with 3 and 1 unrelated",
			args: args{
				t: "json:\"name\" something:\"name2\" json:\"name3\"",
			},
			want: "something:\"name2\" json:\"name3\"",
		},
		{
			name: "Duplicate Test with 3 and 2 unrelated",
			args: args{
				t: "something:\"name1\" json:\"name\" something:\"name2\" json:\"name3\"",
			},
			want: "something:\"name2\" json:\"name3\"",
		},
		{
			name: "Test tag value with leading empty space",
			args: args{
				t: "json:\"name, name2\"",
			},
			want:      "json:\"name, name2\"",
			wantPanic: true,
		},
		{
			name: "Test tag value with trailing empty space",
			args: args{
				t: "json:\"name,name2 \"",
			},
			want:      "json:\"name,name2 \"",
			wantPanic: true,
		},
		{
			name: "Test tag value with space in between",
			args: args{
				t: "gorm:\"unique;not null\"",
			},
			want:      "gorm:\"unique;not null\"",
			wantPanic: false,
		},
		{
			name: "Test mix use of gorm and json tags",
			args: args{
				t: "gorm:\"unique;not null\" json:\"name,name2\"",
			},
			want:      "gorm:\"unique;not null\" json:\"name,name2\"",
			wantPanic: false,
		},
		{
			name: "Test gorm tag with colon",
			args: args{
				t: "gorm:\"type:varchar(63);unique_index\"",
			},
			want:      "gorm:\"type:varchar(63);unique_index\"",
			wantPanic: false,
		},
		{
			name: "Test mix use of gorm and duplicate json tags with colon",
			args: args{
				t: "json:\"name0\" gorm:\"type:varchar(63);unique_index\" json:\"name,name2\"",
			},
			want:      "gorm:\"type:varchar(63);unique_index\" json:\"name,name2\"",
			wantPanic: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				assert.Panics(t, func() { removeDuplicateTags(tt.args.t) }, "The code did not panic")
			} else {
				if got := removeDuplicateTags(tt.args.t); got != tt.want {
					t.Errorf("removeDuplicate() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func Test_containsInvalidSpace(t *testing.T) {
	type args struct {
		valuesString string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test tag value with leading empty space",
			args: args{
				valuesString: "name, name2",
			},
			want: true,
		},
		{
			name: "Test tag value with trailing empty space",
			args: args{
				valuesString: "name ,name2",
			},
			want: true,
		},
		{
			name: "Test tag value with valid empty space in words",
			args: args{
				valuesString: "accept this,name2",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, containsInvalidSpace(tt.args.valuesString), "containsInvalidSpace(%v)", tt.args.valuesString)
		})
	}
}

func Test_splitTagsBySpace(t *testing.T) {
	type args struct {
		tagsString string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "multiple tags, single value",
			args: args{
				tagsString: "json:\"name\" something:\"name2\" json:\"name3\"",
			},
			want: []string{"json:\"name\"", "something:\"name2\"", "json:\"name3\""},
		},
		{
			name: "multiple tag, multiple values",
			args: args{
				tagsString: "json:\"name\" something:\"name2\" json:\"name3,name4\"",
			},
			want: []string{"json:\"name\"", "something:\"name2\"", "json:\"name3,name4\""},
		},
		{
			name: "single tag, single value",
			args: args{
				tagsString: "json:\"name\"",
			},
			want: []string{"json:\"name\""},
		},
		{
			name: "single tag, multiple values",
			args: args{
				tagsString: "json:\"name,name2\"",
			},
			want: []string{"json:\"name,name2\""},
		},
		{
			name: "space in value",
			args: args{
				tagsString: "gorm:\"not nul,name2\"",
			},
			want: []string{"gorm:\"not nul,name2\""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, splitTagsBySpace(tt.args.tagsString), "splitTagsBySpace(%v)", tt.args.tagsString)
		})
	}
}

func TestCustomTemplate(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/gqlgen_custom_model_template.yml")
	require.NoError(t, err)
	require.NoError(t, cfg.Init())
	p := Plugin{
		MutateHook: mutateHook,
		FieldHook:  DefaultFieldMutateHook,
	}
	require.NoError(t, p.MutateConfig(cfg))
}
