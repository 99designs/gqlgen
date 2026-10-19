package tablecodegen

import (
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/validator"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
)

func TestObjectMarshal(t *testing.T) {
	schema := &ast.Schema{Types: map[string]*ast.Definition{
		"Query":     {Name: "Query", Kind: ast.Object},
		"User":      {Name: "User", Kind: ast.Object},
		"Character": {Name: "Character", Kind: ast.Interface},
	}}
	data := newData(&codegen.Data{Config: &config.Config{Schema: schema}})

	require.Equal(
		t,
		"ec.tables.objectUser.Marshal(ctx, ec, sel, &v)",
		data.ObjectMarshal("User", "sel", "&v"),
	)
	require.Equal(
		t,
		"ec.tables.objectQuery.Marshal(ctx, ec, sel, nil)",
		data.ObjectMarshal("Query", "sel", ""),
	)
	require.Equal(t, "_Character(ctx, ec, sel, v)", data.ObjectMarshal("Character", "sel", "v"))
}

func TestTableNames(t *testing.T) {
	ref := func(path, name string) *config.TypeReference {
		pkg := types.NewPackage(path, filepath.Base(path))
		named := types.NewNamed(types.NewTypeName(0, pkg, name, nil), types.Typ[types.String], nil)
		return &config.TypeReference{
			Definition: &ast.Definition{Name: name, Kind: ast.Object},
			GQL:        ast.NonNullNamedType(name, nil),
			GO:         types.NewPointer(named),
		}
	}
	data := newData(&codegen.Data{ReferencedTypes: map[string]*config.TypeReference{
		"post":       ref("github.com/x/app/model", "Post"),
		"user":       ref("github.com/x/app/model", "User"),
		"other user": ref("github.com/x/other/model", "User"),
	}})

	require.Equal(
		t,
		"marshalNPost2ᚖmodelᚐPost",
		data.tableName("marshalNPost2ᚖgithubᚗcomᚋxᚋappᚋmodelᚐPost"),
	)
	// Two packages called model hold a User, so the names of both keep the import path.
	require.Equal(
		t,
		"marshalNUser2ᚖgithubᚗcomᚋxᚋappᚋmodelᚐUser",
		data.tableName("marshalNUser2ᚖgithubᚗcomᚋxᚋappᚋmodelᚐUser"),
	)
	require.Equal(
		t,
		"marshalNUser2ᚖgithubᚗcomᚋxᚋotherᚋmodelᚐUser",
		data.tableName("marshalNUser2ᚖgithubᚗcomᚋxᚋotherᚋmodelᚐUser"),
	)
}

// A list has a typed combinator even when it carries the IsMarshaler of its elements,
// as a list of an enum that marshals itself does; a named slice that marshals itself
// has none.
func TestTableTypedCombinator(t *testing.T) {
	elem := ast.NonNullNamedType("Permission", nil)
	list := &config.TypeReference{
		GO:          types.NewSlice(types.Typ[types.String]),
		GQL:         ast.NonNullListType(elem, nil),
		IsMarshaler: true,
	}
	require.True(t, list.IsSlice())
	require.True(t, tableTypedCombinator(list))

	named := &config.TypeReference{
		GO: types.NewNamed(
			types.NewTypeName(0, nil, "Permissions", nil),
			types.NewSlice(types.Typ[types.String]),
			nil,
		),
		GQL:         ast.NonNullListType(elem, nil),
		IsMarshaler: true,
	}
	require.False(t, named.IsSlice())
	require.False(t, tableTypedCombinator(named))
}

func TestShortTypeIdentifier(t *testing.T) {
	for id, want := range map[string]string{
		"githubᚗcomᚋxᚋappᚋmodelᚐUser":      "modelᚐUser",
		"ᚖgithubᚗcomᚋxᚋappᚋmodelᚐUser":     "ᚖmodelᚐUser",
		"ᚕᚖgithubᚗcomᚋxᚋmyᚑappᚋmodelᚐUser": "ᚕᚖmodelᚐUser",
		"timeᚐTime": "timeᚐTime",
		"ᚖstring":   "ᚖstring",
		"map":       "map",
	} {
		require.Equal(t, want, shortTypeIdentifier(id), id)
	}
}

func TestGroupTables(t *testing.T) {
	var tables []*codegen.Object
	for _, n := range []int{10, 5, 3, 20, 1} {
		tables = append(tables, &codegen.Object{Fields: make([]*codegen.Field, n)})
	}
	var sizes [][]int
	for _, group := range groupTables(tables) {
		var s []int
		for _, o := range group {
			s = append(s, len(o.Fields))
		}
		sizes = append(sizes, s)
	}
	// Tables share a function while their fields fit in tableInitFields; a table with
	// more fields than that gets a function of its own.
	require.Equal(t, [][]int{{10, 5}, {3}, {20}, {1}}, sizes)
}

func TestTableFieldLines(t *testing.T) {
	group := []TableField{
		{Name: "objectUser", Type: "Object"},
		{Name: "objectPost", Type: "Object"},
		{Name: "directiveAuth", Type: "DirectiveDef"},
		{Name: "dirs1", Type: "Dirs"},
	}
	for _, n := range []string{"n1", "n2", "n3", "n4", "n5", "n6", "n7"} {
		group = append(group, TableField{Name: n, Type: "T"})
	}
	require.Equal(t, []TableField{
		// The fields of a type of their own come first, the shortest name first.
		{Name: "dirs1", Type: "Dirs", Section: true},
		{Name: "directiveAuth", Type: "DirectiveDef", Section: true},
		// Then the fields of each shared type, by type, up to six per line.
		{Name: "objectPost, objectUser", Type: "Object", Section: true},
		{Name: "n1, n2, n3, n4, n5, n6", Type: "T"},
		{Name: "n7", Type: "T", Section: true},
	}, tableFieldLines(group))
}

func TestTableType(t *testing.T) {
	empty := types.NewInterfaceType(nil, nil)
	require.Equal(t, "*any", tableType(types.NewPointer(empty)))
	require.Equal(t, "map[string]any", tableType(types.NewMap(types.Typ[types.String], empty)))
	require.Equal(t, "[]string", tableType(types.NewSlice(types.Typ[types.String])))
}

func TestTableFieldArgs(t *testing.T) {
	// A cast keeps the generated unmarshal function, whose In the tables declare.
	ref := &config.TypeReference{
		Definition: &ast.Definition{Name: "Int", Kind: ast.Scalar},
		GQL:        ast.NonNullNamedType("Int", nil),
		GO:         types.Typ[types.Int],
		CastType:   types.Typ[types.Int],
	}
	data := newData(&codegen.Data{
		Config:          &config.Config{},
		ReferencedTypes: map[string]*config.TypeReference{},
	})
	a := &ast.ArgumentDefinition{Name: "a"}
	b := &ast.ArgumentDefinition{Name: "b"}
	c := &ast.ArgumentDefinition{Name: "c"}
	field := &codegen.Field{
		FieldDefinition: &ast.FieldDefinition{
			Name:      "f",
			Arguments: ast.ArgumentDefinitionList{a, b, c},
		},
		// The method of the field takes c and a, in that order, and not b.
		Args: []*codegen.FieldArgument{
			{ArgumentDefinition: c, TypeReference: ref},
			{ArgumentDefinition: a, TypeReference: ref},
		},
	}

	// The arguments are named, in the order of the parameters of the method, and b, which
	// Go does not take, comes last without an In.
	require.Equal(
		t,
		`[]tableArg{{Name: "c", In: t.inUnmarshalNInt2int.In}, {Name: "a", In: t.inUnmarshalNInt2int.In}, {Name: "b"}}`,
		data.TableFieldArgs(field),
	)
}

// The tables give the Go type of an argument or input field that errors name when the
// runtime writes it otherwise from its reflect.Type.
func TestTableGoType(t *testing.T) {
	pkg := types.NewPackage("github.com/x/app/model", "model")
	money := types.NewNamed(types.NewTypeName(0, pkg, "Money", nil), types.Typ[types.Int], nil)
	byteType := types.Universe.Lookup("byte").Type()
	empty := types.NewInterfaceType(nil, nil)
	for _, tc := range []struct {
		typ          types.Type
		reflect, got string
	}{
		{types.NewPointer(money), "*github.com/x/app/model.Money", ""},
		{types.NewSlice(types.Typ[types.Uint8]), "[]uint8", ""},
		{types.NewSlice(byteType), "[]uint8", "[]byte"},
		{types.NewMap(types.Typ[types.String], empty), "map[string]any", "map[string]interface{}"},
		{types.NewMap(types.Typ[types.String], types.Universe.Lookup("any").Type()), "map[string]any", ""},
		{types.NewArray(types.Universe.Lookup("rune").Type(), 2), "[2]int32", "[2]rune"},
		{types.Universe.Lookup("error").Type(), "error", ""},
		// A type the runtime writes in a way tableReflectName does not know gets its name.
		{types.NewStruct(nil, nil), "", "struct{}"},
	} {
		require.Equal(t, tc.reflect, tableReflectName(tc.typ), tc.typ.String())
		require.Equal(t, tc.got, tableGoType(&config.TypeReference{GO: tc.typ}), tc.typ.String())
	}
}

// The functions that table mode generates for objects and inputs do not take the names
// of the functions of the schema's types.
func TestTableFuncNames(t *testing.T) {
	schema := &ast.Schema{Types: map[string]*ast.Definition{
		"User":          {Name: "User", Kind: ast.Object},
		"User_resolve":  {Name: "User_resolve", Kind: ast.Union},
		"User_resolve_": {Name: "User_resolve_", Kind: ast.Interface},
		"Post":          {Name: "Post", Kind: ast.Object},
		"Pair":          {Name: "Pair", Kind: ast.InputObject},
		"Pair_set":      {Name: "Pair_set", Kind: ast.Interface},
	}}
	objects := codegen.Objects{
		{Definition: schema.Types["User"]},
		{Definition: schema.Types["Post"]},
		{Definition: schema.Types["Pair"]},
	}
	d := newData(&codegen.Data{})
	tableFuncNames(schema, objects, d.shared.funcs)
	require.Equal(t, "_User_resolve__", d.TableFunc(objects[0]))
	require.Equal(t, "_Post_resolve", d.TableFunc(objects[1]))
	require.Equal(t, "_Pair_set_", d.TableFunc(objects[2]))
	// An object that a plugin added after Build has the name without underscores.
	require.Equal(
		t,
		"_User_resolve",
		d.TableFunc(&codegen.Object{Definition: schema.Types["User"]}),
	)
}

// A field whose Go type its marshaler does not take is resolved as the functions mode
// resolves it, which returns the value as any.
func TestTableGetExpr(t *testing.T) {
	str := types.NewPointer(types.Typ[types.String])
	d := newData(&codegen.Data{})
	field := func(goType types.Type) *codegen.Field {
		f := &codegen.Field{
			FieldDefinition: &ast.FieldDefinition{Name: "name"},
			TypeReference:   &config.TypeReference{GO: str},
			GoFieldType:     codegen.GoFieldVariable,
			GoReceiverName:  "obj",
			GoFieldName:     "Name",
		}
		d.shared.goTypes[f] = goType
		return f
	}
	require.Equal(t, "obj.Name", d.tableGetExpr(field(str)))
	require.Equal(t, "obj.Name", d.tableGetExpr(field(nil)))
	require.Empty(t, d.tableGetExpr(field(types.NewStruct(nil, nil))))
}

// The Go types of the fields are those of the struct fields and of the results of the
// methods that BuildData binds them to.
func TestTableGoTypes(t *testing.T) {
	pkg := types.NewPackage("github.com/x/app/model", "model")
	str := types.Typ[types.String]
	name := types.NewField(0, pkg, "Name", str, false)
	user := types.NewNamed(
		types.NewTypeName(0, pkg, "User", nil),
		types.NewStruct([]*types.Var{name}, nil),
		nil,
	)
	result := types.NewTuple(types.NewVar(0, pkg, "", types.NewPointer(str)))
	user.AddMethod(types.NewFunc(0, pkg, "Nick", types.NewSignatureType(
		types.NewVar(0, pkg, "u", types.NewPointer(user)), nil, nil, nil, result, false,
	)))
	field := func(name string, kind codegen.GoFieldType) *codegen.Field {
		return &codegen.Field{
			FieldDefinition: &ast.FieldDefinition{Name: name},
			GoFieldType:     kind,
			GoReceiverName:  "obj",
			GoFieldName:     templates.ToGo(name),
		}
	}
	byVar := field("name", codegen.GoFieldVariable)
	byMethod := field("nick", codegen.GoFieldMethod)
	resolver := field("friends", codegen.GoFieldVariable)
	resolver.IsResolver = true
	goTypes := tableGoTypes(codegen.Objects{{
		Definition: &ast.Definition{Name: "User", Kind: ast.Object},
		Type:       user,
		Fields:     []*codegen.Field{byVar, byMethod, resolver},
	}})
	require.Equal(t, map[*codegen.Field]types.Type{
		byVar:    str,
		byMethod: types.NewPointer(str),
	}, goTypes)
}

// The tables are linked to the schema that the code is generated for when plugins
// changed it after it was loaded from the sources.
func TestTableLinkedSchema(t *testing.T) {
	sources := []*ast.Source{
		{Name: "schema.graphql", Input: `
			"""Marks a field."""
			directive @mark on FIELD_DEFINITION
			directive @argOnly on ARGUMENT_DEFINITION
			directive @goTag(key: String!, value: String) on FIELD_DEFINITION
			directive @tagged on SCALAR | FIELD_DEFINITION
			schema { query: Query, mutation: Mutation }
			extend scalar String @tagged
			type Query { a(x: Int = 1): Int, plan: Subscription }
			type Mutation { b: Int }
			type Subscription { name: String }
		`},
		// A built-in source, as inline arguments write.
		{Name: "builtin.graphql", Input: `scalar _Any`, BuiltIn: true},
	}
	load := func() *ast.Schema {
		s, err := gqlparser.LoadSchema(sources...)
		require.NoError(t, err)
		return s
	}
	skipRuntime := func(name string) bool { return name == "goTag" }
	addTo := func(s *ast.Schema, field, directive string) {
		f := s.Query.Fields.ForName(field)
		f.Directives = append(
			f.Directives,
			&ast.Directive{Name: directive, Definition: s.Directives[directive]},
		)
	}

	sdl, err := tableLinkedSchema(load(), sources, skipRuntime)
	require.NoError(t, err)
	require.Empty(t, sdl)

	changed := load()
	addTo(changed, "a", "mark")
	sdl, err = tableLinkedSchema(changed, sources, skipRuntime)
	require.NoError(t, err)
	// The SDL loads back into the changed schema, with the built-in source kept.
	linked, err := gqlparser.LoadSchema(&ast.Source{Name: "linked.graphql", Input: sdl})
	require.NoError(t, err)
	require.NotNil(t, linked.Query.Fields.ForName("a").Directives.ForName("mark"))
	require.NotNil(t, linked.Types["_Any"])
	require.Equal(t, "1", linked.Query.Fields.ForName("a").Arguments.ForName("x").DefaultValue.Raw)
	// The roots are those of the schema definition, not the types called as roots.
	require.Equal(t, "Mutation", linked.Mutation.Name)
	require.Nil(t, linked.Subscription)
	require.Equal(t, ast.Object, linked.Types["Subscription"].Kind)
	// The directives on the types of the prelude that run around fields are kept.
	require.NotNil(t, linked.Types["String"].Directives.ForName("tagged"))

	// Directives that the runtime does not run, and those where their definitions do not
	// allow them, are not written, so that the schema is the same.
	misplaced := load()
	addTo(misplaced, "a", "argOnly")
	addTo(misplaced, "a", "goTag")
	sdl, err = tableLinkedSchema(misplaced, sources, skipRuntime)
	require.NoError(t, err)
	require.Empty(t, sdl)

	// A schema that SDL cannot write fails the generation.
	empty := load()
	empty.Types["Subscription"].Fields = nil
	_, err = tableLinkedSchema(empty, sources, skipRuntime)
	require.ErrorContains(t, err, "table mode cannot link the tables")

	// gqlgen adds a query type with only the fields of introspection to a schema without
	// one, as config.LoadSchema does here. It is written with a field that the tables do
	// not have, as the query type.
	noQuery := []*ast.Source{{Name: "schema.graphql", Input: `type Mutation { a: Int }`}}
	s, err := gqlparser.LoadSchema(noQuery...)
	require.NoError(t, err)
	s.Query = &ast.Definition{
		Kind:   ast.Object,
		Name:   "Query",
		Fields: validator.QueryIntrospectionFields(),
	}
	s.Types["Query"] = s.Query
	sdl, err = tableLinkedSchema(s, noQuery, skipRuntime)
	require.NoError(t, err)
	linked, err = gqlparser.LoadSchema(&ast.Source{Name: "linked.graphql", Input: sdl})
	require.NoError(t, err)
	require.NotNil(t, linked.Query.Fields.ForName(tableEmptyQueryField))
	require.NotNil(t, linked.Query.Fields.ForName("__schema"))
}

// A directive that a plugin puts where its definition does not allow it runs where the
// functions mode runs it: the linked schema allows it there. Where the runtime would then
// run it elsewhere than the functions mode, the generation fails.
func TestTableLinkedSchemaMisplaced(t *testing.T) {
	sources := []*ast.Source{{Name: "schema.graphql", Input: `
		directive @onObject on OBJECT
		directive @inputFieldOnly on INPUT_FIELD_DEFINITION
		input In { v: Int }
		type Query { a(in: In): Int }
	`}}
	load := func() *ast.Schema {
		s, err := gqlparser.LoadSchema(sources...)
		require.NoError(t, err)
		return s
	}
	never := func(string) bool { return false }

	// The functions mode runs a directive declared on OBJECT on a field.
	s := load()
	a := s.Query.Fields.ForName("a")
	a.Directives = append(
		a.Directives,
		&ast.Directive{Name: "onObject", Definition: s.Directives["onObject"]},
	)
	sdl, err := tableLinkedSchema(s, sources, never)
	require.NoError(t, err)
	linked, err := gqlparser.LoadSchema(&ast.Source{Name: "linked.graphql", Input: sdl})
	require.NoError(t, err)
	require.NotNil(t, linked.Query.Fields.ForName("a").Directives.ForName("onObject"))
	require.Equal(t,
		[]ast.DirectiveLocation{ast.LocationObject, ast.LocationFieldDefinition},
		linked.Directives["onObject"].Locations)

	// The functions mode runs a directive declared on INPUT_FIELD_DEFINITION on an input
	// object around the input fields of the input, but not on the input itself, which the
	// runtime would do once the linked schema allows it there.
	s = load()
	in := s.Types["In"]
	in.Directives = append(in.Directives,
		&ast.Directive{Name: "inputFieldOnly", Definition: s.Directives["inputFieldOnly"]})
	_, err = tableLinkedSchema(s, sources, never)
	require.ErrorContains(t, err, "@inputFieldOnly on In")
}

// The directives of the schema itself, which Link does not read, do not keep the schema
// from being linked, even when a directive comes both on the schema and on an extension
// of it, which SDL cannot write as one schema.
func TestTableLinkedSchemaSchemaDirectives(t *testing.T) {
	sources := []*ast.Source{{Name: "schema.graphql", Input: `
		directive @a on SCHEMA
		directive @mark on FIELD_DEFINITION
		schema @a { query: Query }
		extend schema @a
		type Query { a: Int }
	`}}
	s, err := gqlparser.LoadSchema(sources...)
	require.NoError(t, err)
	a := s.Query.Fields.ForName("a")
	a.Directives = append(
		a.Directives,
		&ast.Directive{Name: "mark", Definition: s.Directives["mark"]},
	)
	sdl, err := tableLinkedSchema(s, sources, func(string) bool { return false })
	require.NoError(t, err)
	linked, err := gqlparser.LoadSchema(&ast.Source{Name: "linked.graphql", Input: sdl})
	require.NoError(t, err)
	require.NotNil(t, linked.Query.Fields.ForName("a").Directives.ForName("mark"))
}

// A directive that a plugin puts on a field of a type of the prelude, which SDL cannot
// write, fails the generation rather than not run.
func TestTableLinkedSchemaPreludeFields(t *testing.T) {
	sources := []*ast.Source{{Name: "schema.graphql", Input: `
		directive @mark on FIELD_DEFINITION
		type Query { a: Int }
	`}}
	s, err := gqlparser.LoadSchema(sources...)
	require.NoError(t, err)
	name := s.Types["__Type"].Fields.ForName("name")
	name.Directives = append(
		name.Directives,
		&ast.Directive{Name: "mark", Definition: s.Directives["mark"]},
	)
	_, err = tableLinkedSchema(s, sources, func(string) bool { return false })
	require.ErrorContains(t, err, "__Type.name")
}

// A schema loaded with another value of the prelude is the schema of its sources too.
func TestTableLinkedSchemaPreludeCopy(t *testing.T) {
	prelude := *validator.Prelude
	sources := []*ast.Source{{Name: "schema.graphql", Input: `type Query { a: Int }`}}
	schema, err := validator.LoadSchema(append([]*ast.Source{&prelude}, sources...)...)
	require.NoError(t, err)
	sdl, err := tableLinkedSchema(schema, sources, func(string) bool { return false })
	require.NoError(t, err)
	require.Empty(t, sdl)
}

// The sources may put directives on the types of introspection, which SDL cannot write: the
// tables are linked to the schema of the sources then, unless a plugin changed it.
func TestTableLinkedSchemaIntrospection(t *testing.T) {
	sources := []*ast.Source{{Name: "schema.graphql", Input: `
		directive @seen on OBJECT | FIELD_DEFINITION
		type Query { a: Int }
		extend type __Schema @seen
	`}}
	never := func(string) bool { return false }
	s, err := gqlparser.LoadSchema(sources...)
	require.NoError(t, err)
	sdl, err := tableLinkedSchema(s, sources, never)
	require.NoError(t, err)
	require.Empty(t, sdl)

	s, err = gqlparser.LoadSchema(sources...)
	require.NoError(t, err)
	q := s.Query.Fields.ForName("a")
	q.Directives = append(
		q.Directives,
		&ast.Directive{Name: "seen", Definition: s.Directives["seen"]},
	)
	_, err = tableLinkedSchema(s, sources, never)
	require.ErrorContains(t, err, "@seen on __Schema")
}

// The sources may add fields to the types of introspection, which SDL cannot write either:
// the tables are linked to the schema of the sources, and the generation fails where a
// plugin changes the directives of such a field, or the schema elsewhere.
func TestTableLinkedSchemaIntrospectionFields(t *testing.T) {
	sources := []*ast.Source{{Name: "schema.graphql", Input: `
		directive @a(x: Int) on FIELD_DEFINITION
		type Query { a: Int }
		extend type __Type { extra: String @a(x: 1) plain: String }
	`}}
	never := func(string) bool { return false }
	load := func() *ast.Schema {
		s, err := gqlparser.LoadSchema(sources...)
		require.NoError(t, err)
		return s
	}
	sdl, err := tableLinkedSchema(load(), sources, never)
	require.NoError(t, err)
	require.Empty(t, sdl)

	s := load()
	s.Types["__Type"].Fields.ForName("extra").Directives[0].Arguments[0].Value.Raw = "2"
	_, err = tableLinkedSchema(s, sources, never)
	require.ErrorContains(t, err, "__Type.extra")

	s = load()
	s.Types["__Type"].Fields.ForName("extra").Directives = nil
	q := s.Query.Fields.ForName("a")
	q.Directives = append(q.Directives, &ast.Directive{Name: "a", Definition: s.Directives["a"]})
	_, err = tableLinkedSchema(s, sources, never)
	require.ErrorContains(t, err, "__Type.plain")

	// A plugin that changes the type of a field added to a type of introspection, or the
	// interfaces of the type.
	s = load()
	s.Types["__Type"].Fields.ForName("plain").Type = ast.NonNullNamedType("String", nil)
	_, err = tableLinkedSchema(s, sources, never)
	require.ErrorContains(t, err, "__Type.plain(): String!")
	s = load()
	s.Types["__Type"].Interfaces = []string{"Node"}
	_, err = tableLinkedSchema(s, sources, never)
	require.ErrorContains(t, err, "__Type implements Node")
}

// The functions mode reads the interfaces and unions of an object from the map of the
// schema, which a plugin may leave out of step with the definitions, from which the runtime
// reads them: the generation fails then. The maps that only the code that both modes share
// reads may differ.
func TestTableLinkedSchemaMembers(t *testing.T) {
	sources := []*ast.Source{{Name: "schema.graphql", Input: `
		interface Node { id: ID! }
		type User implements Node { id: ID! }
		type Robot { id: ID! }
		union Thing = User | Robot
		type Query { user: User thing: Thing }
	`}}
	never := func(string) bool { return false }
	load := func() *ast.Schema {
		s, err := gqlparser.LoadSchema(sources...)
		require.NoError(t, err)
		return s
	}
	sdl, err := tableLinkedSchema(load(), sources, never)
	require.NoError(t, err)
	require.Empty(t, sdl)

	s := load()
	s.Types["User"].Interfaces = nil
	_, err = tableLinkedSchema(s, sources, never)
	require.ErrorContains(t, err, "User")

	s = load()
	s.Types["Thing"].Types = []string{"User"}
	_, err = tableLinkedSchema(s, sources, never)
	require.ErrorContains(t, err, "Robot")

	s = load()
	s.AddImplements("Robot", s.Types["Node"])
	_, err = tableLinkedSchema(s, sources, never)
	require.ErrorContains(t, err, "Robot")

	s = load()
	s.AddPossibleType("Node", s.Types["Robot"])
	sdl, err = tableLinkedSchema(s, sources, never)
	require.NoError(t, err)
	require.Empty(t, sdl)

	// A plugin that changes both, which the tables are linked to.
	s = load()
	s.Types["Robot"].Interfaces = append(s.Types["Robot"].Interfaces, "Node")
	s.AddImplements("Robot", s.Types["Node"])
	s.AddPossibleType("Node", s.Types["Robot"])
	sdl, err = tableLinkedSchema(s, sources, never)
	require.NoError(t, err)
	require.Contains(t, sdl, "type Robot implements Node")

	// The members of a union may be types of introspection, which SDL does not write.
	sources = []*ast.Source{{Name: "schema.graphql", Input: `
		type User { id: ID! }
		union Meta = User | __Type
		type Query { meta: Meta }
	`}}
	sdl, err = tableLinkedSchema(load(), sources, never)
	require.NoError(t, err)
	require.Empty(t, sdl)
	s = load()
	s.Types["Meta"].Types = []string{"User"}
	_, err = tableLinkedSchema(s, sources, never)
	require.ErrorContains(t, err, "__Type")
}

// A plugin that puts a directive twice on a field makes the functions mode run it twice,
// so the linked schema lets the definition repeat, which SDL requires.
func TestTableLinkedSchemaRepeated(t *testing.T) {
	sources := []*ast.Source{{Name: "schema.graphql", Input: `
		directive @auth on FIELD_DEFINITION
		type Query { a: Int @auth }
	`}}
	s, err := gqlparser.LoadSchema(sources...)
	require.NoError(t, err)
	a := s.Query.Fields.ForName("a")
	a.Directives = append(
		a.Directives,
		&ast.Directive{Name: "auth", Definition: s.Directives["auth"]},
	)
	sdl, err := tableLinkedSchema(s, sources, func(string) bool { return false })
	require.NoError(t, err)
	linked, err := gqlparser.LoadSchema(&ast.Source{Name: "linked.graphql", Input: sdl})
	require.NoError(t, err)
	require.Len(t, linked.Query.Fields.ForName("a").Directives.ForNames("auth"), 2)
}

// The functions mode reads where a directive runs from the definition that the schema had
// when it was loaded, which a plugin may replace with one declared elsewhere.
func TestTableLinkedSchemaReplacedDefinition(t *testing.T) {
	sources := []*ast.Source{{Name: "schema.graphql", Input: `
		directive @d on SCALAR | FIELD_DEFINITION
		scalar S @d
		type Query { s: S }
	`}}
	s, err := gqlparser.LoadSchema(sources...)
	require.NoError(t, err)
	d := *s.Directives["d"]
	d.Locations = append(slices.Clone(d.Locations), ast.LocationInputObject)
	s.Directives["d"] = &d
	// The functions mode runs @d around Query.s, which the runtime would not do with the
	// replaced definition.
	_, err = tableLinkedSchema(s, sources, func(string) bool { return false })
	require.ErrorContains(t, err, "@d on S")
}

// The generated code embeds the sources under its directory from their files, which a
// program that runs gqlgen may have changed in memory: the tables are linked to the schema
// in memory then.
func TestTableSources(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "schema.graphql"),
		[]byte(`type Query { a: Int }`), 0o600))
	sources := []*ast.Source{
		{
			Name:  "schema.graphql",
			Input: "directive @d on FIELD_DEFINITION\ntype Query { a: Int @d }",
		},
		{Name: "other.graphql", Input: "type Other { b: Int }"},
	}
	augmented := []codegen.AugmentedSource{
		{RelativePath: "schema.graphql", Embeddable: true},
		{RelativePath: "../other.graphql"},
	}
	got := tableSources(sources, augmented, dir)
	require.Equal(t, []*ast.Source{
		{Name: "schema.graphql", Input: `type Query { a: Int }`},
		sources[1],
	}, got)

	s, err := gqlparser.LoadSchema(sources...)
	require.NoError(t, err)
	sdl, err := tableLinkedSchema(s, got, func(string) bool { return false })
	require.NoError(t, err)
	require.NotEmpty(t, sdl)
}

// The function that table mode keeps for a reference to an object assigns the value to the
// Go type in which the functions mode passes it, which converts a value of another type
// that can be assigned to it, as the resolve function of the object asserts that type.
func TestTableChecksObject(t *testing.T) {
	pkg := types.NewPackage("github.com/x/app/model", "model")
	model := types.NewMap(types.Typ[types.String], types.NewInterfaceType(nil, nil))
	profile := types.NewNamed(types.NewTypeName(0, pkg, "Profile", nil), model, nil)
	def := &ast.Definition{Name: "Profile", Kind: ast.Object}
	d := newData(&codegen.Data{
		Config: &config.Config{
			Schema: &ast.Schema{Types: map[string]*ast.Definition{"Profile": def}},
		},
		Objects:         codegen.Objects{{Definition: def, Type: model}},
		ReferencedTypes: map[string]*config.TypeReference{},
	})
	require.False(t, d.tableChecksObject("Profile", model))
	require.True(t, d.tableChecksObject("Profile", profile))
}

// The runtime derives the Out of the pointers to an object and of the lists of them,
// unless a list that the tables declare refers to the Out of its elements.
func TestTableDerived(t *testing.T) {
	pkg := types.NewPackage("github.com/x/app/model", "model")
	user := types.NewNamed(types.NewTypeName(0, pkg, "User", nil), types.NewStruct(nil, nil), nil)
	def := &ast.Definition{Name: "User", Kind: ast.Object}
	ptr := &config.TypeReference{
		Definition: def,
		GQL:        ast.NonNullNamedType("User", nil),
		GO:         types.NewPointer(user),
		Target:     user,
	}
	list := *ptr
	list.GQL = ast.NonNullListType(ptr.GQL, nil)
	list.GO = types.NewSlice(ptr.GO)
	nested := *ptr
	nested.GQL = ast.NonNullListType(list.GQL, nil)
	nested.GO = types.NewSlice(list.GO)
	data := func(refs ...*config.TypeReference) *Data {
		d := newData(&codegen.Data{
			Config:          &config.Config{},
			Objects:         codegen.Objects{{Definition: def, Type: user}},
			ReferencedTypes: map[string]*config.TypeReference{},
		})
		for _, ref := range refs {
			d.ReferencedTypes[ref.GQL.String()] = ref
		}
		return d
	}

	d := data(ptr, &list)
	require.True(t, d.tableDerived(ptr, true))
	require.True(t, d.tableDerived(&list, true))
	require.Panics(t, func() { d.TableOut(ptr) })

	// The list of a nested list is marshaled by its typed marshaler, which marshals the
	// elements with their Out: the tables declare both, and the nested list.
	d = data(ptr, &list, &nested)
	require.False(t, d.tableDerived(ptr, true))
	require.False(t, d.tableDerived(&list, true))
	require.False(t, d.tableDerived(&nested, true))
	require.Equal(t, "t.marshalNUser2ᚖmodelᚐUser", d.TableOut(ptr))
}

// The code that a plugin generated calls an unmarshaler, which table mode keeps typed, where
// it would otherwise build it without a type parameter, which the code cannot call.
func TestAddPluginCall(t *testing.T) {
	list := &config.TypeReference{
		Definition: &ast.Definition{Name: "String", Kind: ast.Scalar},
		GQL:        ast.NonNullListType(ast.NonNullNamedType("String", nil), nil),
		GO:         types.NewSlice(types.Typ[types.String]),
	}
	data := func() *Data {
		return newData(&codegen.Data{
			Config:          &config.Config{},
			ReferencedTypes: map[string]*config.TypeReference{list.GQL.String(): list},
		})
	}

	d := data()
	require.Panics(t, func() { d.UnmarshalCall(list) })

	d = data()
	d.addPluginCall(list.UnmarshalFunc())
	require.Equal(t, "ec.tables.unmarshalNString2ᚕstringᚄ.Unmarshal", d.UnmarshalCall(list))
}

// newData returns the Data of table mode for data, as Build does without checking data.
func newData(data *codegen.Data) *Data {
	return &Data{
		Data: data,
		shared: &shared{
			funcs:   map[*codegen.Object]string{},
			goTypes: map[*codegen.Field]types.Type{},
		},
	}
}
