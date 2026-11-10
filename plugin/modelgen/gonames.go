package modelgen

import (
	"fmt"
	"go/types"

	"github.com/99designs/gqlgen/codegen/templates"
)

// member is a field or method that models.gotpl generates on a Go type.
type member struct {
	goName string
	source string // for errors, such as `field "id"`
}

// checkGoNames rejects a build whose generated types would declare a field
// or method name twice, which would otherwise fail only when the generated
// code compiles.
//
// Ensures: nil, or an error naming the type, both members, and their Go name.
func checkGoNames(b *ModelBuild) error {
	for _, it := range b.Interfaces {
		if err := checkMembers(it.Name, interfaceMembers(it)); err != nil {
			return err
		}
	}
	for _, model := range b.Models {
		if err := checkMembers(model.Name, modelMembers(model)); err != nil {
			return err
		}
	}
	return nil
}

func checkMembers(typeName string, members []member) error {
	seen := make(map[string]string, len(members))
	for _, m := range members {
		if other, ok := seen[m.goName]; ok {
			return fmt.Errorf(
				"type %q: %s and %s both have Go name %s; "+
					"rename one with @goField(name: ...) or gqlgen.yml fieldName",
				typeName, other, m.source, m.goName,
			)
		}
		seen[m.goName] = m.source
	}
	return nil
}

// interfaceMembers returns the getters models.gotpl declares on it. Its
// Is<Interface> markers are left out: they can clash only with each other,
// and each name is listed once.
func interfaceMembers(it *Interface) []member {
	var members []member
	for _, field := range it.Fields {
		members = append(members, getter(it.Name, field))
	}
	return members
}

// modelMembers returns the fields and methods models.gotpl declares on
// model.
//
// Requires: model.Implementations is set (implementations).
func modelMembers(model *Object) []member {
	var members []member
	for _, field := range model.Fields {
		if field.GoName != "" {
			members = append(members, member{field.GoName, fmt.Sprintf("field %q", field.Name)})
			continue
		}
		if name := embeddedName(field.Type); name != "" {
			members = append(members, member{name, "embedded " + name})
		}
	}
	for _, impl := range model.Implementations {
		members = append(members, marker(impl.Interface))
		for _, g := range impl.Getters {
			members = append(members, member{"Get" + g.Name, g.describe()})
		}
	}
	return members
}

func marker(interfaceName string) member {
	return member{
		"Is" + templates.ToGoModelName(interfaceName),
		fmt.Sprintf("the method marking interface %q", interfaceName),
	}
}

func getter(interfaceName string, field *Field) member {
	return member{
		"Get" + field.GoName,
		fmt.Sprintf("the getter for %s.%s", interfaceName, field.Name),
	}
}

// embeddedName returns the field name Go gives an anonymous field of type t.
func embeddedName(t types.Type) string {
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	if named, ok := t.(*types.Named); ok {
		return named.Obj().Name()
	}
	return ""
}
