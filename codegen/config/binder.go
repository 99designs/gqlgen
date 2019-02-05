package config

import (
	"fmt"
	"go/types"
	"regexp"
	"strings"

	"github.com/99designs/gqlgen/internal/code"
	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/ast"
	"golang.org/x/tools/go/packages"
)

// Binder connects graphql types to golang types using static analysis
type Binder struct {
	pkgs       []*packages.Package
	schema     *ast.Schema
	cfg        *Config
	References []*TypeReference
}

func (c *Config) NewBinder(s *ast.Schema) (*Binder, error) {
	pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadTypes | packages.LoadSyntax}, c.Models.ReferencedPackages()...)
	if err != nil {
		return nil, err
	}

	return &Binder{
		pkgs:   pkgs,
		schema: s,
		cfg:    c,
	}, nil
}

func (b *Binder) FindType(pkgName string, typeName string) (types.Type, error) {
	obj, err := b.FindObject(pkgName, typeName)
	if err != nil {
		return nil, err
	}

	if fun, isFunc := obj.(*types.Func); isFunc {
		return fun.Type().(*types.Signature).Params().At(0).Type(), nil
	}
	return obj.Type(), nil
}

func (b *Binder) getPkg(find string) *packages.Package {
	for _, p := range b.pkgs {
		if normalizeVendor(find) == normalizeVendor(p.PkgPath) {
			return p
		}
	}
	return nil
}

var MapType = types.NewMap(types.Typ[types.String], types.NewInterfaceType(nil, nil).Complete())
var InterfaceType = types.NewInterfaceType(nil, nil)

func (b *Binder) FindUserObject(name string) (types.Type, error) {
	userEntry, ok := b.cfg.Models[name]
	if !ok {
		return nil, fmt.Errorf(name + " not found")
	}

	if userEntry.Model == "map[string]interface{}" {
		return MapType, nil
	}

	if userEntry.Model == "interface{}" {
		return InterfaceType, nil
	}

	pkgName, typeName := code.PkgAndType(userEntry.Model)
	if pkgName == "" {
		return nil, fmt.Errorf("missing package name for %s", name)
	}

	obj, err := b.FindObject(pkgName, typeName)
	if err != nil {
		return nil, err
	}

	return obj.Type(), nil
}

func (b *Binder) FindObject(pkgName string, typeName string) (types.Object, error) {
	if pkgName == "" {
		return nil, fmt.Errorf("package cannot be nil")
	}
	fullName := typeName
	if pkgName != "" {
		fullName = pkgName + "." + typeName
	}

	pkg := b.getPkg(pkgName)
	if pkg == nil {
		return nil, errors.Errorf("required package was not loaded: %s", fullName)
	}

	for astNode, def := range pkg.TypesInfo.Defs {
		// only look at defs in the top scope
		if def == nil || def.Parent() == nil || def.Parent() != pkg.Types.Scope() {
			continue
		}

		if astNode.Name == typeName || astNode.Name == "Marshal"+typeName {
			return def, nil
		}
	}

	return nil, errors.Errorf("unable to find type %s\n", fullName)
}

var modsRegex = regexp.MustCompile(`^(\*|\[\])*`)

func normalizeVendor(pkg string) string {
	modifiers := modsRegex.FindAllString(pkg, 1)[0]
	pkg = strings.TrimPrefix(pkg, modifiers)
	parts := strings.Split(pkg, "/vendor/")
	return modifiers + parts[len(parts)-1]
}

// TypeReference is used by args and field types. The Definition can refer to both input and output types.
type TypeReference struct {
	Definition  *ast.Definition
	GQL         *ast.Type
	GO          types.Type
	Marshaler   *types.Func // When using external marshalling functions this will point to the Marshal function
	Unmarshaler *types.Func // When using external marshalling functions this will point to the Unmarshal function
}

func (t TypeReference) IsPtr() bool {
	_, isPtr := t.GO.(*types.Pointer)
	return isPtr
}

func (t TypeReference) IsSlice() bool {
	_, isSlice := t.GO.(*types.Slice)
	return isSlice
}

func (t TypeReference) IsNamed() bool {
	_, isSlice := t.GO.(*types.Named)
	return isSlice
}

func (t TypeReference) IsScalar() bool {
	return t.Definition.Kind == ast.Scalar
}

func (t TypeReference) SelfMarshalling() bool {
	it := t.GO
	if ptr, isPtr := it.(*types.Pointer); isPtr {
		it = ptr.Elem()
	}
	namedType, ok := it.(*types.Named)
	if !ok {
		return false
	}

	for i := 0; i < namedType.NumMethods(); i++ {
		switch namedType.Method(i).Name() {
		case "MarshalGQL":
			return true
		}
	}
	return false
}

func (t TypeReference) NeedsUnmarshaler() bool {
	if t.Definition == nil {
		panic(errors.New("Definition missing for " + t.GQL.Name()))
	}
	return t.Definition.IsInputType()
}

func (t TypeReference) NeedsMarshaler() bool {
	if t.Definition == nil {
		panic(errors.New("Definition missing for " + t.GQL.Name()))
	}
	return t.Definition.Kind != ast.InputObject
}

func (b *Binder) PushRef(ret *TypeReference) {
	b.References = append(b.References, ret)
}

func (b *Binder) TypeReference(schemaType *ast.Type) (ret *TypeReference, err error) {
	var pkgName, typeName string
	def := b.schema.Types[schemaType.Name()]
	defer func() {
		if err == nil && ret != nil {
			b.PushRef(ret)
		}
	}()

	if userEntry, ok := b.cfg.Models[schemaType.Name()]; ok && userEntry.Model != "" {
		if userEntry.Model == "map[string]interface{}" {
			return &TypeReference{
				Definition: def,
				GQL:        schemaType,
				GO:         MapType,
			}, nil
		}

		if userEntry.Model == "interface{}" {
			return &TypeReference{
				Definition: def,
				GQL:        schemaType,
				GO:         InterfaceType,
			}, nil
		}

		pkgName, typeName = code.PkgAndType(userEntry.Model)
		if pkgName == "" {
			return nil, fmt.Errorf("missing package name for %s", schemaType.Name())
		}

	} else {
		pkgName = "github.com/99designs/gqlgen/graphql"
		typeName = "String"
	}

	ref := &TypeReference{
		Definition: def,
		GQL:        schemaType,
	}

	obj, err := b.FindObject(pkgName, typeName)
	if err != nil {
		return nil, err
	}

	if fun, isFunc := obj.(*types.Func); isFunc {
		ref.GO = fun.Type().(*types.Signature).Params().At(0).Type()
		ref.Marshaler = fun
		ref.Unmarshaler = types.NewFunc(0, fun.Pkg(), "Unmarshal"+typeName, nil)
	} else {
		ref.GO = obj.Type()
	}

	if namedType, ok := ref.GO.(*types.Named); ok && ref.Unmarshaler == nil {
		hasUnmarshal := false
		for i := 0; i < namedType.NumMethods(); i++ {
			switch namedType.Method(i).Name() {
			case "UnmarshalGQL":
				hasUnmarshal = true
			}
		}

		// Special case to reference generated unmarshal functions
		if !hasUnmarshal {
			ref.Unmarshaler = types.NewFunc(0, b.cfg.Exec.Pkg(), "ec.unmarshalInput"+schemaType.Name(), nil)
		}
	}

	ref.GO = b.CopyModifiersFromAst(schemaType, def.Kind != ast.Interface, ref.GO)

	return ref, nil
}

func (b *Binder) CopyModifiersFromAst(t *ast.Type, usePtr bool, base types.Type) types.Type {
	if t.Elem != nil {
		return types.NewSlice(b.CopyModifiersFromAst(t.Elem, usePtr, base))
	}

	if !t.NonNull && usePtr {
		return types.NewPointer(base)
	}

	return base
}
