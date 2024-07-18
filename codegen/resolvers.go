package codegen

// import (
// 	"fmt"
// 	"go/types"
// 	"strconv"
// 	"strings"
// 	"unicode"

// 	"github.com/vektah/gqlparser/v2/ast"
// 	"golang.org/x/text/cases"
// 	"golang.org/x/text/language"

// 	"github.com/99designs/gqlgen/codegen/config"
// )

// type GoFieldType int

// const (
// 	GoFieldUndefined GoFieldType = iota
// 	GoFieldMethod
// 	GoFieldVariable
// 	GoFieldMap
// )

// type ResolverObject struct {
// 	Name      string
// 	Resolvers []*Resolver
// }

// func (b *builder) buildResolver(typ *ast.Definition) (*ResolverObject, error) {
// 	dirs, err := b.getDirectives(typ.Directives)
// 	if err != nil {
// 		return nil, fmt.Errorf("%s: %w", typ.Name, err)
// 	}
// 	caser := cases.Title(language.English, cases.NoLower)
// 	obj := &ResolverObject{
// 		Definition:               typ,
// 		Root:                     b.Config.IsRoot(typ),
// 		DisableConcurrency:       typ == b.Schema.Mutation,
// 		Stream:                   typ == b.Schema.Subscription,
// 		Directives:               dirs,
// 		PointersInUnmarshalInput: b.Config.ReturnPointersInUnmarshalInput,
// 		ResolverInterface: types.NewNamed(
// 			types.NewTypeName(0, b.Config.Exec.Pkg(), caser.String(typ.Name)+"ResolverObject", nil),
// 			nil,
// 			nil,
// 		),
// 	}

// 	if !obj.Root {
// 		goObject, err := b.Binder.DefaultUserObject(typ.Name)
// 		if err != nil {
// 			return nil, err
// 		}
// 		obj.Type = goObject
// 	}

// 	for _, intf := range b.Schema.GetImplements(typ) {
// 		obj.Implements = append(obj.Implements, b.Schema.Types[intf.Name])
// 	}

// 	for _, field := range typ.Fields {
// 		if strings.HasPrefix(field.Name, "__") {
// 			continue
// 		}

// 		var f *Field
// 		f, err = b.buildField(obj, field)
// 		if err != nil {
// 			return nil, err
// 		}

// 		obj.Fields = append(obj.Fields, f)
// 	}

// 	return obj, nil
// }

// func (o *ResolverObject) Reference() types.Type {
// 	if config.IsNilable(o.Type) {
// 		return o.Type
// 	}
// 	return types.NewPointer(o.Type)
// }

// type ResolverObjects []*ResolverObject

// func (o *ResolverObject) IsConcurrent() bool {
// 	for _, f := range o.Fields {
// 		if f.IsConcurrent() {
// 			return true
// 		}
// 	}
// 	return false
// }

// func (o *ResolverObject) IsReserved() bool {
// 	return strings.HasPrefix(o.Definition.Name, "__")
// }

// func (o *ResolverObject) IsMap() bool {
// 	return o.Type == config.MapType
// }

// func (o *ResolverObject) Description() string {
// 	return o.Definition.Description
// }

// func (o *ResolverObject) HasField(name string) bool {
// 	for _, f := range o.Fields {
// 		if f.Name == name {
// 			return true
// 		}
// 	}

// 	return false
// }

// func (os ResolverObjects) ByName(name string) *ResolverObject {
// 	for i, o := range os {
// 		if strings.EqualFold(o.Definition.Name, name) {
// 			return os[i]
// 		}
// 	}
// 	return nil
// }

// func ucFirst(s string) string {
// 	if s == "" {
// 		return ""
// 	}

// 	r := []rune(s)
// 	r[0] = unicode.ToUpper(r[0])
// 	return string(r)
// }
