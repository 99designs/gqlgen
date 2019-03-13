package config

import (
	"fmt"
	"go/token"
	"go/types"

	"github.com/99designs/gqlgen/codegen/templates"
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

	for _, p := range pkgs {
		for _, e := range p.Errors {
			if e.Kind == packages.ListError {
				return nil, p.Errors[0]
			}
		}
	}

	return &Binder{
		pkgs:   pkgs,
		schema: s,
		cfg:    c,
	}, nil
}

func (b *Binder) TypePosition(typ types.Type) token.Position {
	named, isNamed := typ.(*types.Named)
	if !isNamed {
		return token.Position{
			Filename: "unknown",
		}
	}

	return b.ObjectPosition(named.Obj())
}

func (b *Binder) ObjectPosition(typ types.Object) token.Position {
	if typ == nil {
		return token.Position{
			Filename: "unknown",
		}
	}
	pkg := b.getPkg(typ.Pkg().Path())
	return pkg.Fset.Position(typ.Pos())
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
		if code.NormalizeVendor(find) == code.NormalizeVendor(p.PkgPath) {
			return p
		}
	}
	return nil
}

var MapType = types.NewMap(types.Typ[types.String], types.NewInterface(nil, nil).Complete())
var InterfaceType = types.NewInterface(nil, nil)

func (b *Binder) DefaultUserObject(name string) (types.Type, error) {
	models := b.cfg.Models[name].Model
	if len(models) == 0 {
		return nil, fmt.Errorf(name + " not found in typemap")
	}

	if models[0] == "map[string]interface{}" {
		return MapType, nil
	}

	if models[0] == "interface{}" {
		return InterfaceType, nil
	}

	pkgName, typeName := code.PkgAndType(models[0])
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

	// function based marshalers take precedence
	for astNode, def := range pkg.TypesInfo.Defs {
		// only look at defs in the top scope
		if def == nil || def.Parent() == nil || def.Parent() != pkg.Types.Scope() {
			continue
		}

		if astNode.Name == "Marshal"+typeName {
			return def, nil
		}
	}

	// then look for types directly
	for astNode, def := range pkg.TypesInfo.Defs {
		// only look at defs in the top scope
		if def == nil || def.Parent() == nil || def.Parent() != pkg.Types.Scope() {
			continue
		}

		if astNode.Name == typeName {
			return def, nil
		}
	}

	return nil, errors.Errorf("unable to find type %s\n", fullName)
}

func (b *Binder) PointerTo(ref *TypeReference) *TypeReference {
	newRef := &TypeReference{
		GO:          types.NewPointer(ref.GO),
		GQL:         ref.GQL,
		Definition:  ref.Definition,
		Unmarshaler: ref.Unmarshaler,
		Marshaler:   ref.Marshaler,
	}

	b.References = append(b.References, newRef)
	return newRef
}

// TypeReference is used by args and field types. The Definition can refer to both input and output types.
type TypeReference struct {
	Definition  *ast.Definition
	GQL         *ast.Type
	GO          types.Type
	Marshaler   *types.Func // When using external marshalling functions this will point to the Marshal function
	Unmarshaler *types.Func // When using external marshalling functions this will point to the Unmarshal function
}

func (ref *TypeReference) Elem() *TypeReference {
	if p, isPtr := ref.GO.(*types.Pointer); isPtr {
		return &TypeReference{
			GO:          p.Elem(),
			GQL:         ref.GQL,
			Definition:  ref.Definition,
			Unmarshaler: ref.Unmarshaler,
			Marshaler:   ref.Marshaler,
		}
	}

	if s, isSlice := ref.GO.(*types.Slice); isSlice {
		return &TypeReference{
			GO:          s.Elem(),
			GQL:         ref.GQL.Elem,
			Definition:  ref.Definition,
			Unmarshaler: ref.Unmarshaler,
			Marshaler:   ref.Marshaler,
		}
	}
	return nil
}

func (t *TypeReference) IsPtr() bool {
	_, isPtr := t.GO.(*types.Pointer)
	return isPtr
}

func (t *TypeReference) IsNilable() bool {
	_, isPtr := t.GO.(*types.Pointer)
	_, isMap := t.GO.(*types.Map)
	_, isInterface := t.GO.(*types.Interface)
	return isPtr || isMap || isInterface
}

func (t *TypeReference) IsSlice() bool {
	_, isSlice := t.GO.(*types.Slice)
	return isSlice
}

func (t *TypeReference) IsNamed() bool {
	_, isSlice := t.GO.(*types.Named)
	return isSlice
}

func (t *TypeReference) IsStruct() bool {
	_, isStruct := t.GO.Underlying().(*types.Struct)
	return isStruct
}

func (t *TypeReference) IsScalar() bool {
	return t.Definition.Kind == ast.Scalar
}

func (t *TypeReference) HasIsZero() bool {
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
		case "IsZero":
			return true
		}
	}
	return false
}

func (t *TypeReference) SelfMarshalling() bool {
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

func (t *TypeReference) SelfUnmarshalling() bool {
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
		case "UnmarshalGQL":
			return true
		}
	}
	return false
}

func (t *TypeReference) UniquenessKey() string {
	var nullability = "O"
	if t.GQL.NonNull {
		nullability = "N"
	}

	return nullability + t.Definition.Name + "2" + templates.TypeIdentifier(t.GO)
}

func (t *TypeReference) MarshalFunc() string {
	if t.Definition == nil {
		panic(errors.New("Definition missing for " + t.GQL.Name()))
	}

	if t.Definition.Kind == ast.InputObject {
		return ""
	}

	return "marshal" + t.UniquenessKey()
}

func (t *TypeReference) UnmarshalFunc() string {
	if t.Definition == nil {
		panic(errors.New("Definition missing for " + t.GQL.Name()))
	}

	if !t.Definition.IsInputType() {
		return ""
	}

	return "unmarshal" + t.UniquenessKey()
}

func (b *Binder) PushRef(ret *TypeReference) {
	b.References = append(b.References, ret)
}

func isMap(t types.Type) bool {
	if t == nil {
		return true
	}
	_, ok := t.(*types.Map)
	return ok
}

func isIntf(t types.Type) bool {
	if t == nil {
		return true
	}
	_, ok := t.(*types.Interface)
	return ok
}

func (b *Binder) TypeReference(schemaType *ast.Type, bindTarget types.Type) (ret *TypeReference, err error) {
	var pkgName, typeName string
	def := b.schema.Types[schemaType.Name()]
	defer func() {
		if err == nil && ret != nil {
			b.PushRef(ret)
		}
	}()

	if len(b.cfg.Models[schemaType.Name()].Model) == 0 {
		return nil, fmt.Errorf("%s was not found", schemaType.Name())
	}

	for _, model := range b.cfg.Models[schemaType.Name()].Model {
		if model == "map[string]interface{}" {
			if !isMap(bindTarget) {
				continue
			}
			return &TypeReference{
				Definition: def,
				GQL:        schemaType,
				GO:         MapType,
			}, nil
		}

		if model == "interface{}" {
			if !isIntf(bindTarget) {
				continue
			}
			return &TypeReference{
				Definition: def,
				GQL:        schemaType,
				GO:         InterfaceType,
			}, nil
		}

		pkgName, typeName = code.PkgAndType(model)
		if pkgName == "" {
			return nil, fmt.Errorf("missing package name for %s", schemaType.Name())
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

		ref.GO = b.CopyModifiersFromAst(schemaType, def.Kind != ast.Interface, ref.GO)

		if bindTarget != nil {
			if err = code.CompatibleTypes(ref.GO, bindTarget); err != nil {
				continue
			}
			ref.GO = bindTarget
		}

		return ref, nil
	}

	return nil, fmt.Errorf("%s has type compatible with %s", schemaType.Name(), bindTarget.String())
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
