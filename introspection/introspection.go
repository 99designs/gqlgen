package introspection

import (
	"fmt"
	"sort"

	"github.com/neelance/graphql-go/internal/common"
	"github.com/neelance/graphql-go/internal/schema"
)

type Schema struct {
	Schema *schema.Schema
}

func (r *Schema) Types() []*Type {
	var names []string
	for name := range r.Schema.Types {
		names = append(names, name)
	}
	sort.Strings(names)

	var l []*Type
	for _, name := range names {
		l = append(l, &Type{r.Schema.Types[name]})
	}
	return l
}

func (r *Schema) QueryType() *Type {
	t, ok := r.Schema.EntryPoints["query"]
	if !ok {
		return nil
	}
	return &Type{t}
}

func (r *Schema) MutationType() *Type {
	t, ok := r.Schema.EntryPoints["mutation"]
	if !ok {
		return nil
	}
	return &Type{t}
}

func (r *Schema) SubscriptionType() *Type {
	t, ok := r.Schema.EntryPoints["subscription"]
	if !ok {
		return nil
	}
	return &Type{t}
}

func (r *Schema) Directives() []*Directive {
	return []*Directive{
		&Directive{
			name:        "skip",
			description: "Directs the executor to skip this field or fragment when the `if` argument is true.",
			locations:   []string{"FIELD", "FRAGMENT_SPREAD", "INLINE_FRAGMENT"},
			args: []*InputValue{
				&InputValue{&common.InputValue{
					Name: "if",
					Desc: "Skipped when true.",
					Type: &common.NonNull{OfType: r.Schema.Types["Boolean"]},
				}},
			},
		},
		&Directive{
			name:        "include",
			description: "Directs the executor to include this field or fragment only when the `if` argument is true.",
			locations:   []string{"FIELD", "FRAGMENT_SPREAD", "INLINE_FRAGMENT"},
			args: []*InputValue{
				&InputValue{&common.InputValue{
					Name: "if",
					Desc: "Included when true.",
					Type: &common.NonNull{OfType: r.Schema.Types["Boolean"]},
				}},
			},
		},
	}
}

type Type struct {
	Typ common.Type
}

func (r *Type) Kind() string {
	return r.Typ.Kind()
}

func (r *Type) Name() *string {
	if named, ok := r.Typ.(schema.NamedType); ok {
		name := named.TypeName()
		return &name
	}
	return nil
}

func (r *Type) Description() *string {
	if named, ok := r.Typ.(schema.NamedType); ok {
		desc := named.Description()
		if desc == "" {
			return nil
		}
		return &desc
	}
	return nil
}

func (r *Type) Fields(args *struct{ IncludeDeprecated bool }) *[]*Field {
	var fields map[string]*schema.Field
	var fieldOrder []string
	switch t := r.Typ.(type) {
	case *schema.Object:
		fields = t.Fields
		fieldOrder = t.FieldOrder
	case *schema.Interface:
		fields = t.Fields
		fieldOrder = t.FieldOrder
	default:
		return nil
	}

	l := make([]*Field, len(fieldOrder))
	for i, name := range fieldOrder {
		l[i] = &Field{fields[name]}
	}
	return &l
}

func (r *Type) Interfaces() *[]*Type {
	t, ok := r.Typ.(*schema.Object)
	if !ok {
		return nil
	}

	l := make([]*Type, len(t.Interfaces))
	for i, intf := range t.Interfaces {
		l[i] = &Type{intf}
	}
	return &l
}

func (r *Type) PossibleTypes() *[]*Type {
	var possibleTypes []*schema.Object
	switch t := r.Typ.(type) {
	case *schema.Interface:
		possibleTypes = t.PossibleTypes
	case *schema.Union:
		possibleTypes = t.PossibleTypes
	default:
		return nil
	}

	l := make([]*Type, len(possibleTypes))
	for i, intf := range possibleTypes {
		l[i] = &Type{intf}
	}
	return &l
}

func (r *Type) EnumValues(args *struct{ IncludeDeprecated bool }) *[]*EnumValue {
	t, ok := r.Typ.(*schema.Enum)
	if !ok {
		return nil
	}

	l := make([]*EnumValue, len(t.Values))
	for i, v := range t.Values {
		l[i] = &EnumValue{v}
	}
	return &l
}

func (r *Type) InputFields() *[]*InputValue {
	t, ok := r.Typ.(*schema.InputObject)
	if !ok {
		return nil
	}

	l := make([]*InputValue, len(t.FieldOrder))
	for i, name := range t.FieldOrder {
		l[i] = &InputValue{t.Fields[name]}
	}
	return &l
}

func (r *Type) OfType() *Type {
	switch t := r.Typ.(type) {
	case *common.List:
		return &Type{t.OfType}
	case *common.NonNull:
		return &Type{t.OfType}
	default:
		return nil
	}
}

type Field struct {
	field *schema.Field
}

func (r *Field) Name() string {
	return r.field.Name
}

func (r *Field) Description() *string {
	if r.field.Desc == "" {
		return nil
	}
	return &r.field.Desc
}

func (r *Field) Args() []*InputValue {
	l := make([]*InputValue, len(r.field.Args.FieldOrder))
	for i, name := range r.field.Args.FieldOrder {
		l[i] = &InputValue{r.field.Args.Fields[name]}
	}
	return l
}

func (r *Field) Type() *Type {
	return &Type{r.field.Type}
}

func (r *Field) IsDeprecated() bool {
	return false
}

func (r *Field) DeprecationReason() *string {
	return nil
}

type InputValue struct {
	value *common.InputValue
}

func (r *InputValue) Name() string {
	return r.value.Name
}

func (r *InputValue) Description() *string {
	if r.value.Desc == "" {
		return nil
	}
	return &r.value.Desc
}

func (r *InputValue) Type() *Type {
	return &Type{r.value.Type}
}

func (r *InputValue) DefaultValue() *string {
	if r.value.Default == nil {
		return nil
	}
	s := fmt.Sprint(r.value.Default)
	return &s
}

type EnumValue struct {
	value *schema.EnumValue
}

func (r *EnumValue) Name() string {
	return r.value.Name
}

func (r *EnumValue) Description() *string {
	if r.value.Desc == "" {
		return nil
	}
	return &r.value.Desc
}

func (r *EnumValue) IsDeprecated() bool {
	return false
}

func (r *EnumValue) DeprecationReason() *string {
	return nil
}

type Directive struct {
	name        string
	description string
	locations   []string
	args        []*InputValue
}

func (r *Directive) Name() string {
	return r.name
}

func (r *Directive) Description() *string {
	return &r.description
}

func (r *Directive) Locations() []string {
	return r.locations
}

func (r *Directive) Args() []*InputValue {
	return r.args
}
