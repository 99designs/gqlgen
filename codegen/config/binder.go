package config

import (
	"errors"
	"fmt"
	"go/token"
	"go/types"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
	"golang.org/x/tools/go/packages"

	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/99designs/gqlgen/internal/code"
)

var ErrTypeNotFound = errors.New("unable to find type")

// Binder connects graphql types to golang types using static analysis
type Binder struct {
	pkgs        *code.Packages
	schema      *ast.Schema
	cfg         *Config
	tctx        *types.Context
	References  []*TypeReference
	SawInvalid  bool
	objectCache map[string]map[string]types.Object
}

func (c *Config) NewBinder() *Binder {
	return &Binder{
		pkgs:   c.Packages,
		schema: c.Schema,
		cfg:    c,
	}
}

func (b *Binder) TypePosition(typ types.Type) token.Position {
	named, isNamed := code.Unalias(typ).(*types.Named)
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
	pkg := b.pkgs.Load(typ.Pkg().Path())
	return pkg.Fset.Position(typ.Pos())
}

func (b *Binder) FindTypeFromName(name string) (types.Type, error) {
	pkgName, typeName := code.PkgAndType(name)
	return b.FindType(pkgName, typeName)
}

func (b *Binder) FindType(pkgName, typeName string) (types.Type, error) {
	if pkgName == "" {
		if typeName == "map[string]any" || typeName == "map[string]interface{}" {
			return MapType, nil
		}

		if typeName == "any" || typeName == "interface{}" {
			return InterfaceType, nil
		}
	}

	obj, err := b.FindObject(pkgName, typeName)
	if err != nil {
		return nil, err
	}

	t := code.Unalias(obj.Type())
	if _, isFunc := obj.(*types.Func); isFunc {
		return code.Unalias(t.(*types.Signature).Params().At(0).Type()), nil
	}
	return t, nil
}

func (b *Binder) InstantiateType(orig types.Type, targs []types.Type) (types.Type, error) {
	if b.tctx == nil {
		b.tctx = types.NewContext()
	}

	return types.Instantiate(b.tctx, orig, targs, false)
}

var (
	MapType       = types.NewMap(types.Typ[types.String], types.NewInterfaceType(nil, nil).Complete())
	InterfaceType = types.NewInterfaceType(nil, nil)
)

func (b *Binder) DefaultUserObject(name string) (types.Type, error) {
	models := b.cfg.Models[name].Model
	if len(models) == 0 {
		return nil, fmt.Errorf("%s not found in typemap", name)
	}

	if models[0] == "map[string]any" || models[0] == "map[string]interface{}" {
		return MapType, nil
	}

	if models[0] == "any" || models[0] == "interface{}" {
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

	return code.Unalias(obj.Type()), nil
}

func (b *Binder) FindObject(pkgName, typeName string) (types.Object, error) {
	if pkgName == "" {
		return nil, fmt.Errorf("package cannot be nil in FindObject for type: %s", typeName)
	}

	pkg := b.pkgs.LoadWithTypes(pkgName)
	if pkg == nil {
		err := b.pkgs.Errors()
		if err != nil {
			return nil, fmt.Errorf("package could not be loaded: %s.%s: %w", pkgName, typeName, err)
		}
		return nil, fmt.Errorf("required package was not loaded: %s.%s", pkgName, typeName)
	}

	if b.objectCache == nil {
		b.objectCache = make(map[string]map[string]types.Object, b.pkgs.Count())
	}

	defsIndex, ok := b.objectCache[pkgName]
	if !ok {
		defsIndex = indexDefs(pkg)
		b.objectCache[pkgName] = defsIndex
	}

	// function based marshalers take precedence
	if val, ok := defsIndex["Marshal"+typeName]; ok {
		return val, nil
	}

	if val, ok := defsIndex[typeName]; ok {
		return val, nil
	}

	return nil, fmt.Errorf("%w: %s.%s", ErrTypeNotFound, pkgName, typeName)
}

func indexDefs(pkg *packages.Package) map[string]types.Object {
	res := make(map[string]types.Object)

	scope := pkg.Types.Scope()
	for astNode, def := range pkg.TypesInfo.Defs {
		// only look at defs in the top scope
		if def == nil {
			continue
		}
		parent := def.Parent()
		if parent == nil || parent != scope {
			continue
		}

		if _, ok := res[astNode.Name]; !ok {
			// The above check may not be really needed, it is only here to have a consistent behavior with
			// previous implementation of FindObject() function which only honored the first inclusion of a def.
			// If this is still needed, we can consider something like sync.Map.LoadOrStore() to avoid two lookups.
			res[astNode.Name] = def
		}
	}

	return res
}

func (b *Binder) PointerTo(ref *TypeReference) *TypeReference {
	newRef := *ref
	newRef.GO = types.NewPointer(ref.GO)
	b.References = append(b.References, &newRef)
	return &newRef
}

// TypeReference is used by args and field types. The Definition can refer to both input and output types.
type TypeReference struct {
	Definition               *ast.Definition
	GQL                      *ast.Type
	GO                       types.Type  // Type of the field being bound. Could be a pointer or a value type of Target.
	Target                   types.Type  // The actual type that we know how to bind to. May require pointer juggling when traversing to fields.
	CastType                 types.Type  // Before calling marshalling functions cast from/to this base type
	Marshaler                *types.Func // When using external marshalling functions this will point to the Marshal function
	Unmarshaler              *types.Func // When using external marshalling functions this will point to the Unmarshal function
	IsMarshaler              bool        // Does the type implement graphql.Marshaler and graphql.Unmarshaler
	IsJSONMarshaler          bool        // Does the type implement json.Marshaler and json.Unmarshaler
	IsOmittable              bool        // Is the type wrapped with Omittable
	IsContext                bool        // Is the Marshaler/Unmarshaller the context version; applies to either the method or interface variety.
	PointersInUnmarshalInput bool        // Inverse values and pointers in return.
	IsRoot                   bool        // Is the type a root level definition such as Query, Mutation or Subscription
	EnumValues               []EnumValueReference
}

func (ref *TypeReference) Elem() *TypeReference {
	if p, isPtr := ref.GO.(*types.Pointer); isPtr {
		newRef := *ref
		newRef.GO = p.Elem()
		return &newRef
	}

	if ref.IsSlice() {
		newRef := *ref
		newRef.GO = ref.GO.(*types.Slice).Elem()
		newRef.GQL = ref.GQL.Elem
		return &newRef
	}
	return nil
}

func (ref *TypeReference) IsPtr() bool {
	_, isPtr := ref.GO.(*types.Pointer)
	return isPtr
}

// fix for https://github.com/golang/go/issues/31103 may make it possible to remove this (may still be useful)
func (ref *TypeReference) IsPtrToPtr() bool {
	if p, isPtr := ref.GO.(*types.Pointer); isPtr {
		_, isPtr := p.Elem().(*types.Pointer)
		return isPtr
	}
	return false
}

func (ref *TypeReference) IsNilable() bool {
	return IsNilable(ref.GO)
}

func (ref *TypeReference) IsSlice() bool {
	_, isSlice := ref.GO.(*types.Slice)
	return ref.GQL.Elem != nil && isSlice
}

func (ref *TypeReference) IsPtrToSlice() bool {
	if ref.IsPtr() {
		_, isPointerToSlice := ref.GO.(*types.Pointer).Elem().(*types.Slice)
		return isPointerToSlice
	}
	return false
}

func (ref *TypeReference) IsPtrToIntf() bool {
	if ref.IsPtr() {
		_, isPointerToInterface := types.Unalias(ref.GO.(*types.Pointer).Elem()).(*types.Interface)
		return isPointerToInterface
	}
	return false
}

func (ref *TypeReference) IsNamed() bool {
	_, ok := ref.GO.(*types.Named)
	return ok
}

func (ref *TypeReference) IsStruct() bool {
	_, ok := ref.GO.Underlying().(*types.Struct)
	return ok
}

func (ref *TypeReference) IsScalar() bool {
	return ref.Definition.Kind == ast.Scalar
}

func (ref *TypeReference) IsMap() bool {
	return ref.GO == MapType
}

func (ref *TypeReference) UniquenessKey() string {
	nullability := "O"
	if ref.GQL.NonNull {
		nullability = "N"
	}

	elemNullability := ""
	if ref.GQL.Elem != nil && ref.GQL.Elem.NonNull {
		// Fix for #896
		elemNullability = "ᚄ"
	}
	return nullability + ref.Definition.Name + "2" + templates.TypeIdentifier(ref.GO) + elemNullability
}

func (ref *TypeReference) MarshalFunc() string {
	if ref.Definition == nil {
		panic(errors.New("Definition missing for " + ref.GQL.Name()))
	}

	if ref.Definition.Kind == ast.InputObject {
		return ""
	}

	return "marshal" + ref.UniquenessKey()
}

func (ref *TypeReference) UnmarshalFunc() string {
	if ref.Definition == nil {
		panic(errors.New("Definition missing for " + ref.GQL.Name()))
	}

	if !ref.Definition.IsInputType() {
		return ""
	}

	return "unmarshal" + ref.UniquenessKey()
}

func (ref *TypeReference) IsTargetNilable() bool {
	return IsNilable(ref.Target)
}

func (ref *TypeReference) HasEnumValues() bool {
	return len(ref.EnumValues) > 0
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
	_, ok := types.Unalias(t).(*types.Interface)
	return ok
}

func unwrapOmittable(t types.Type) (types.Type, bool) {
	if t == nil {
		return nil, false
	}
	named, ok := t.(*types.Named)
	if !ok {
		return t, false
	}
	if named.Origin().String() != "github.com/99designs/gqlgen/graphql.Omittable[T any]" {
		return t, false
	}
	return named.TypeArgs().At(0), true
}

func (b *Binder) TypeReference(schemaType *ast.Type, bindTarget types.Type) (ret *TypeReference, err error) {
	if bindTarget != nil {
		bindTarget = code.Unalias(bindTarget)
	}
	if innerType, ok := unwrapOmittable(bindTarget); ok {
		if schemaType.NonNull {
			return nil, fmt.Errorf("%s is wrapped with Omittable but non-null", schemaType.Name())
		}

		ref, err := b.TypeReference(schemaType, innerType)
		if err != nil {
			return nil, err
		}

		ref.IsOmittable = true
		return ref, err
	}

	if !isValid(bindTarget) {
		b.SawInvalid = true
		return nil, fmt.Errorf("%s has an invalid type", schemaType.Name())
	}

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
		if model == "map[string]any" || model == "map[string]interface{}" {
			if !isMap(bindTarget) {
				continue
			}
			return &TypeReference{
				Definition: def,
				GQL:        schemaType,
				GO:         MapType,
				IsRoot:     b.cfg.IsRoot(def),
			}, nil
		}

		if model == "any" || model == "interface{}" {
			if !isIntf(bindTarget) {
				continue
			}
			return &TypeReference{
				Definition: def,
				GQL:        schemaType,
				GO:         InterfaceType,
				IsRoot:     b.cfg.IsRoot(def),
			}, nil
		}

		pkgName, typeName = code.PkgAndType(model)
		if pkgName == "" {
			return nil, fmt.Errorf("missing package name for %s", schemaType.Name())
		}

		ref := &TypeReference{
			Definition: def,
			GQL:        schemaType,
			IsRoot:     b.cfg.IsRoot(def),
		}

		obj, err := b.FindObject(pkgName, typeName)
		if err != nil {
			return nil, err
		}
		t := code.Unalias(obj.Type())
		if values := b.enumValues(def); len(values) > 0 {
			err = b.enumReference(ref, obj, values)
			if err != nil {
				return nil, err
			}
		} else if fun, isFunc := obj.(*types.Func); isFunc {
			ref.GO = code.Unalias(t.(*types.Signature).Params().At(0).Type())
			ref.IsContext = code.Unalias(t.(*types.Signature).Results().At(0).Type()).String() == "github.com/99designs/gqlgen/graphql.ContextMarshaler"
			ref.Marshaler = fun
			ref.Unmarshaler = types.NewFunc(0, fun.Pkg(), "Unmarshal"+typeName, nil)
		} else if hasMethod(t, "MarshalGQLContext") && hasMethod(t, "UnmarshalGQLContext") {
			ref.GO = t
			ref.IsContext = true
			ref.IsMarshaler = true
		} else if hasMethod(t, "MarshalGQL") && hasMethod(t, "UnmarshalGQL") {
			ref.GO = t
			ref.IsMarshaler = true
		} else if hasMethod(t, "MarshalJSON") && hasMethod(t, "UnmarshalJSON") {
			ref.GO = t
			ref.IsJSONMarshaler = true
		} else if underlying := basicUnderlying(t); def.IsLeafType() && underlying != nil && underlying.Kind() == types.String {
			// TODO delete before v1. Backwards compatibility case for named types wrapping strings (see #595)

			ref.GO = t
			ref.CastType = underlying

			underlyingRef, err := b.TypeReference(&ast.Type{NamedType: "String"}, nil)
			if err != nil {
				return nil, err
			}

			ref.Marshaler = underlyingRef.Marshaler
			ref.Unmarshaler = underlyingRef.Unmarshaler
		} else {
			ref.GO = t
		}

		ref.Target = ref.GO
		ref.GO = b.CopyModifiersFromAst(schemaType, ref.GO)

		if bindTarget != nil {
			if err = code.CompatibleTypes(ref.GO, bindTarget); err != nil {
				continue
			}
			ref.GO = bindTarget
		}

		ref.PointersInUnmarshalInput = b.cfg.ReturnPointersInUnmarshalInput

		return ref, nil
	}

	return nil, fmt.Errorf("%s is incompatible with %s", schemaType.Name(), bindTarget.String())
}

func isValid(t types.Type) bool {
	basic, isBasic := t.(*types.Basic)
	if !isBasic {
		return true
	}
	return basic.Kind() != types.Invalid
}

func (b *Binder) CopyModifiersFromAst(t *ast.Type, base types.Type) types.Type {
	base = types.Unalias(base)
	if t.Elem != nil {
		child := b.CopyModifiersFromAst(t.Elem, base)
		if _, isStruct := child.Underlying().(*types.Struct); isStruct && !b.cfg.OmitSliceElementPointers {
			child = types.NewPointer(child)
		}
		return types.NewSlice(child)
	}

	var isInterface bool
	if named, ok := base.(*types.Named); ok {
		_, isInterface = named.Underlying().(*types.Interface)
	}

	if !isInterface && !IsNilable(base) && !t.NonNull {
		return types.NewPointer(base)
	}

	return base
}

func IsNilable(t types.Type) bool {
	// Note that we use types.Unalias rather than code.Unalias here
	// because we want to always check the underlying type.
	// code.Unalias only unwraps aliases in Go 1.23
	t = types.Unalias(t)
	if namedType, isNamed := t.(*types.Named); isNamed {
		return IsNilable(namedType.Underlying())
	}
	_, isPtr := t.(*types.Pointer)
	_, isNilableMap := t.(*types.Map)
	_, isInterface := t.(*types.Interface)
	_, isSlice := t.(*types.Slice)
	_, isChan := t.(*types.Chan)
	return isPtr || isNilableMap || isInterface || isSlice || isChan
}

func hasMethod(it types.Type, name string) bool {
	if ptr, isPtr := it.(*types.Pointer); isPtr {
		it = ptr.Elem()
	}
	namedType, ok := it.(*types.Named)
	if !ok {
		return false
	}

	for i := 0; i < namedType.NumMethods(); i++ {
		if namedType.Method(i).Name() == name {
			return true
		}
	}
	return false
}

func basicUnderlying(it types.Type) *types.Basic {
	it = types.Unalias(it)
	if ptr, isPtr := it.(*types.Pointer); isPtr {
		it = types.Unalias(ptr.Elem())
	}
	namedType, ok := it.(*types.Named)
	if !ok {
		return nil
	}

	if basic, ok := namedType.Underlying().(*types.Basic); ok {
		return basic
	}

	return nil
}

type EnumValueReference struct {
	Definition *ast.EnumValueDefinition
	Object     types.Object
}

func (b *Binder) enumValues(def *ast.Definition) map[string]EnumValue {
	if def.Kind != ast.Enum {
		return nil
	}

	if strings.HasPrefix(def.Name, "__") {
		return nil
	}

	model, ok := b.cfg.Models[def.Name]
	if !ok {
		return nil
	}

	return model.EnumValues
}

func (b *Binder) enumReference(ref *TypeReference, obj types.Object, values map[string]EnumValue) error {
	if len(ref.Definition.EnumValues) != len(values) {
		return fmt.Errorf("not all enum values are binded for %v", ref.Definition.Name)
	}

	t := code.Unalias(obj.Type())
	if fn, ok := t.(*types.Signature); ok {
		ref.GO = code.Unalias(fn.Params().At(0).Type())
	} else {
		ref.GO = t
	}

	str, err := b.TypeReference(&ast.Type{NamedType: "String"}, nil)
	if err != nil {
		return err
	}

	ref.Marshaler = str.Marshaler
	ref.Unmarshaler = str.Unmarshaler
	ref.EnumValues = make([]EnumValueReference, 0, len(values))

	for _, value := range ref.Definition.EnumValues {
		v, ok := values[value.Name]
		if !ok {
			return fmt.Errorf("enum value not found for: %v, of enum: %v", value.Name, ref.Definition.Name)
		}

		pkgName, typeName := code.PkgAndType(v.Value)
		if pkgName == "" {
			return fmt.Errorf("missing package name for %v", value.Name)
		}

		valueObj, err := b.FindObject(pkgName, typeName)
		if err != nil {
			return err
		}

		valueTyp := code.Unalias(valueObj.Type())
		if !types.AssignableTo(valueTyp, ref.GO) {
			return fmt.Errorf("wrong type: %v, for enum value: %v, expected type: %v, of enum: %v",
				valueTyp, value.Name, ref.GO, ref.Definition.Name)
		}

		switch valueObj.(type) {
		case *types.Const, *types.Var:
			ref.EnumValues = append(ref.EnumValues, EnumValueReference{
				Definition: value,
				Object:     valueObj,
			})
		default:
			return fmt.Errorf("unsupported enum value for: %v, of enum: %v, only const and var allowed",
				value.Name, ref.Definition.Name)
		}
	}

	return nil
}
