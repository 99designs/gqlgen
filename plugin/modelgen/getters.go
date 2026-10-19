package modelgen

import (
	"fmt"
	"go/types"
	"strings"
	"text/template"

	"github.com/99designs/gqlgen/codegen/templates"
)

// Implementation is an interface or union a model implements, with the
// getters the model needs for it.
type Implementation struct {
	Interface string // GraphQL name
	Getters   []*Getter
}

// Getter is a method that returns an interface field from a model.
type Getter struct {
	Name        string     // the method is Get<Name>: the interface field's Go name
	Field       string     // the model field it returns, by Go name
	Type        types.Type // the interface field's type
	Description string
	PointerOp   string // "&" or "*" applied to the field to match Type, or ""
	Slice       bool   // copied element by element, since a []T is not a []I

	interfaceName string
	source        *Field // the interface field
}

type getterKey struct {
	model *Object
	field *Field // the interface field
}

// implementations derives, for every model in b, the interfaces it
// implements and the getters they need, in Implements and field order.
// A getter returns the model field with the interface field's GraphQL
// name, or one promoted from an embedded struct.
//
// Ensures: one Implementation per Implements entry; at most one getter per
// Go name; and an error for each getter that can't be generated correctly:
// two interface fields with one Go name, one field with two Go types, or no
// field to return. Such a getter is kept as gqlgen generated it before, so
// custom templates see the same getters.
func implementations(b *ModelBuild) (map[*Object][]*Implementation, []error) {
	interfaces := make(map[string]*Interface, len(b.Interfaces))
	for _, it := range b.Interfaces {
		interfaces[it.Name] = it
	}

	result := make(map[*Object][]*Implementation, len(b.Models))
	var errs []error
	for _, model := range b.Models {
		impls, modelErrs := modelImplementations(model, interfaces)
		result[model] = impls
		errs = append(errs, modelErrs...)
	}
	return result, errs
}

func modelImplementations(
	model *Object,
	interfaces map[string]*Interface,
) ([]*Implementation, []error) {
	impls := make([]*Implementation, 0, len(model.Implements))
	byName := make(map[string]*Getter)
	var errs []error
	for _, name := range model.Implements {
		impl := &Implementation{Interface: name}
		impls = append(impls, impl)
		it := interfaces[name]
		if it == nil {
			continue // bound or unknown: no getters, as before
		}
		for _, field := range it.Fields {
			g, ok := newGetter(model, it.Name, field)
			if prev, dup := byName[g.Name]; dup {
				if err := sameGetter(model.Name, prev, g); err != nil {
					errs = append(errs, err)
				}
				continue
			}
			if !ok {
				errs = append(errs, fmt.Errorf(
					"type %q: %s has no field to return; generate field %q on %q, "+
						"or set omit_getters",
					model.Name, g.describe(), field.Name, model.Name,
				))
			}
			byName[g.Name] = g
			impl.Getters = append(impl.Getters, g)
		}
	}
	return impls, errs
}

// newGetter builds the getter for an interface field on model. ok is false
// when no field can supply it; the getter then reads the interface field's
// Go name, as gqlgen did before.
func newGetter(model *Object, interfaceName string, field *Field) (g *Getter, ok bool) {
	g = &Getter{
		Name:          field.GoName,
		Field:         field.GoName,
		Type:          field.Type,
		Description:   field.Description,
		interfaceName: interfaceName,
		source:        field,
	}
	_, interfacePtr := field.Type.(*types.Pointer)
	// A promoted field is assumed to have the interface field's Go name and
	// type, as the base structs @goEmbedInterface generates do.
	structPtr := interfacePtr
	target, embeds := findField(model, field.Name)
	if target != nil {
		g.Field = target.GoName
		_, structPtr = target.Type.(*types.Pointer)
	}
	switch {
	case interfacePtr && !structPtr:
		g.PointerOp = "&"
	case !interfacePtr && structPtr:
		g.PointerOp = "*"
	}
	_, g.Slice = field.Type.(*types.Slice)
	return g, target != nil || embeds
}

// findField returns model's field with GraphQL name name. embeds reports
// that model embeds a struct, which may hold the field instead.
func findField(model *Object, name string) (field *Field, embeds bool) {
	for _, f := range model.Fields {
		if f.Name == name {
			return f, false
		}
		embeds = embeds || f.GoName == ""
	}
	return nil, embeds
}

// sameGetter reports whether b, for a later interface, can share the getter
// a already generated.
func sameGetter(modelName string, a, b *Getter) error {
	if a.source.Name != b.source.Name {
		return fmt.Errorf(
			"type %q: %s and %s both have Go name Get%s; "+
				"rename one with @goField(name: ...) or gqlgen.yml fieldName",
			modelName, a.describe(), b.describe(), a.Name,
		)
	}
	// Compared as strings: gqlgen builds a fresh types.Named for every
	// reference to a generated type.
	if aType, bType := types.TypeString(
		a.Type,
		nil,
	), types.TypeString(
		b.Type,
		nil,
	); aType != bType {
		return fmt.Errorf(
			"type %q: %s returns %s but %s returns %s; give the interface fields one type",
			modelName, a.describe(), aType, b.describe(), bType,
		)
	}
	return nil
}

func (g *Getter) describe() string {
	return fmt.Sprintf("the getter for %s.%s", g.interfaceName, g.source.Name)
}

// render returns g as a method of the Go type receiver, with goType the
// generated file's spelling of g.Type.
func (g *Getter) render(receiver, goType string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "func (this %s) Get%s() %s {", receiver, g.Name, goType)
	if !g.Slice {
		fmt.Fprintf(&sb, " return %sthis.%s }", g.PointerOp, g.Field)
		return sb.String()
	}
	fmt.Fprintf(&sb, "\n\tif this.%s == nil { return nil }\n", g.Field)
	fmt.Fprintf(&sb, "\tinterfaceSlice := make(%s, 0, len(this.%s))\n", goType, g.Field)
	fmt.Fprintf(&sb, "\tfor _, concrete := range this.%s { ", g.Field)
	fmt.Fprintf(&sb, "interfaceSlice = append(interfaceSlice, %sconcrete) }\n", g.PointerOp)
	sb.WriteString("\treturn interfaceSlice\n}")
	return sb.String()
}

// templateFuncs returns the functions models.gotpl and custom templates call.
//
// Requires: every model's Implementations is set.
func templateFuncs(b *ModelBuild) template.FuncMap {
	interfaces := make(map[string]*Interface, len(b.Interfaces))
	for _, it := range b.Interfaces {
		interfaces[it.Name] = it
	}
	getters := make(map[getterKey]*Getter)
	for _, model := range b.Models {
		for _, impl := range model.Implementations {
			for _, g := range impl.Getters {
				getters[getterKey{model, g.source}] = g
			}
		}
	}
	return template.FuncMap{
		"getter": func(model *Object, g *Getter, goType string) string {
			return g.render(templates.ToGoModelName(model.Name), goType)
		},
		// For custom templates, which predate Implementations.
		"getInterfaceByName": func(name string) *Interface { return interfaces[name] },
		// For custom templates: the getter for an interface field, or "" if
		// the model has none or shares one generated for another interface.
		// Unlike getter, it reads the global import set to spell the type.
		"generateGetter": func(model *Object, field *Field) string {
			g := getters[getterKey{model, field}]
			if g == nil {
				return ""
			}
			return g.render(
				templates.ToGoModelName(model.Name),
				templates.CurrentImports.LookupType(g.Type),
			)
		},
	}
}
