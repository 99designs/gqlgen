package introspection

import (
	"fmt"
	"sort"

	"github.com/neelance/graphql-go/internal/common"
	"github.com/neelance/graphql-go/internal/schema"
)

type Schema struct {
	schema *schema.Schema
}

// WrapSchema is only used internally.
func WrapSchema(schema *schema.Schema) *Schema {
	return &Schema{schema}
}

func (r *Schema) Types() []*Type {
	var names []string
	for name := range r.schema.Types {
		names = append(names, name)
	}
	sort.Strings(names)

	l := make([]*Type, len(names))
	for i, name := range names {
		l[i] = &Type{r.schema.Types[name]}
	}
	return l
}

func (r *Schema) Directives() []*Directive {
	var names []string
	for name := range r.schema.Directives {
		names = append(names, name)
	}
	sort.Strings(names)

	l := make([]*Directive, len(names))
	for i, name := range names {
		l[i] = &Directive{r.schema.Directives[name]}
	}
	return l
}

func (r *Schema) QueryType() *Type {
	t, ok := r.schema.EntryPoints["query"]
	if !ok {
		return nil
	}
	return &Type{t}
}

func (r *Schema) MutationType() *Type {
	t, ok := r.schema.EntryPoints["mutation"]
	if !ok {
		return nil
	}
	return &Type{t}
}

func (r *Schema) SubscriptionType() *Type {
	t, ok := r.schema.EntryPoints["subscription"]
	if !ok {
		return nil
	}
	return &Type{t}
}

type Type struct {
	typ common.Type
}

// WrapType is only used internally.
func WrapType(typ common.Type) *Type {
	return &Type{typ}
}

func (r *Type) Kind() string {
	return r.typ.Kind()
}

func (r *Type) Name() *string {
	if named, ok := r.typ.(schema.NamedType); ok {
		name := named.TypeName()
		return &name
	}
	return nil
}

func (r *Type) Description() *string {
	if named, ok := r.typ.(schema.NamedType); ok {
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
	switch t := r.typ.(type) {
	case *schema.Object:
		fields = t.Fields
		fieldOrder = t.FieldOrder
	case *schema.Interface:
		fields = t.Fields
		fieldOrder = t.FieldOrder
	default:
		return nil
	}

	var l []*Field
	for _, name := range fieldOrder {
		f := fields[name]
		if _, ok := f.Directives["deprecated"]; !ok || args.IncludeDeprecated {
			l = append(l, &Field{f})
		}
	}
	return &l
}

func (r *Type) Interfaces() *[]*Type {
	t, ok := r.typ.(*schema.Object)
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
	switch t := r.typ.(type) {
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
	t, ok := r.typ.(*schema.Enum)
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
	t, ok := r.typ.(*schema.InputObject)
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
	switch t := r.typ.(type) {
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
	_, ok := r.field.Directives["deprecated"]
	return ok
}

func (r *Field) DeprecationReason() *string {
	args, ok := r.field.Directives["deprecated"]
	if !ok {
		return nil
	}
	reason := args["reason"].(string)
	return &reason
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
	directive *schema.Directive
}

func (r *Directive) Name() string {
	return r.directive.Name
}

func (r *Directive) Description() *string {
	if r.directive.Desc == "" {
		return nil
	}
	return &r.directive.Desc
}

func (r *Directive) Locations() []string {
	return r.directive.Locs
}

func (r *Directive) Args() []*InputValue {
	l := make([]*InputValue, len(r.directive.ArgOrder))
	for i, name := range r.directive.ArgOrder {
		l[i] = &InputValue{r.directive.Args[name]}
	}
	return l
}
