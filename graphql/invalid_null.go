package graphql

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

// ErrInvalidNull matches any non-null violation through [errors.Is]. For the
// details, recover [InvalidNullError] with [errors.AsType].
var ErrInvalidNull = errors.New("invalid null")

// NullSource names the Go construct a null came from — the part neither the
// schema nor the response path can tell you.
type NullSource string

const (
	// NullSourceUnknown is used when there is no field context to inspect.
	NullSourceUnknown NullSource = "unknown"
	// NullSourceResolver is a resolver method that returned nil.
	NullSourceResolver NullSource = "resolver"
	// NullSourceModelMethod is a model method that returned nil.
	NullSourceModelMethod NullSource = "model_method"
	// NullSourceModelField is a nil model struct field with no resolver.
	NullSourceModelField NullSource = "model_field"
	// NullSourceMarshaler is a marshaler that mapped a non-nil value to null.
	NullSourceMarshaler NullSource = "marshaler"
	// NullSourceRuntimeNonNull is a nullable field raised by [MarkNonNull].
	NullSourceRuntimeNonNull NullSource = "runtime_non_null"
)

// InvalidNullError reports that a position the schema declares non-null resolved to
// null. Presenters receive it wrapped in a [gqlerror.Error] and recover it with
// [errors.AsType]. Build them with [AddInvalidNullError].
type InvalidNullError struct {
	// Object is the concrete GraphQL type owning the field, which Path cannot
	// give for a field reached through an interface or union.
	Object string
	// Field is the schema field name, falling back to the response alias.
	Field string
	// DeclaredType is the SDL type that forbade the null ("InnerObject!"), or
	// the element type when ListElement is set.
	DeclaredType string
	// Source is the Go construct the null came from.
	Source NullSource
	// ListElement reports a nil list element rather than a nil field value.
	// Path then ends in the index.
	ListElement bool
	// Path is the response path of the null, including list indices.
	Path ast.Path
}

// SchemaCoordinate returns the field's schema coordinate, "Object.field", as
// defined by the GraphQL specification. It returns "" unless both halves are
// known, since a bare field name is not a coordinate.
func (e *InvalidNullError) SchemaCoordinate() string {
	if e.Object == "" || e.Field == "" {
		return ""
	}
	return e.Object + "." + e.Field
}

// Attrs reports the violation as log attributes, so middleware that collects
// Attrs() []slog.Attr along an error chain can record it without knowing this
// type. Empty values are omitted rather than logged as blanks.
func (e *InvalidNullError) Attrs() []slog.Attr {
	attrs := make([]slog.Attr, 0, 5)
	if coord := e.SchemaCoordinate(); coord != "" {
		attrs = append(attrs, slog.String("gql.field", coord))
	}
	if e.DeclaredType != "" {
		attrs = append(attrs, slog.String("gql.type", e.DeclaredType))
	}
	if e.Source != "" {
		attrs = append(attrs, slog.String("gql.null_source", string(e.Source)))
	}
	if e.ListElement {
		attrs = append(attrs, slog.Bool("gql.list_element", true))
	}
	if path := e.Path.String(); path != "" {
		attrs = append(attrs, slog.String("gql.path", path))
	}
	return attrs
}

// Is reports [ErrInvalidNull] so errors.Is matches any non-null violation. Unwrap
// would be wrong: the sentinel caused nothing, so it would show up as a link to
// anything walking the chain, and would occupy the slot a real cause may need.
func (e *InvalidNullError) Is(target error) bool { return target == ErrInvalidNull }

// Error names the position, the type that forbade the null, and where the null
// came from. The wording is not API and will change; match the type with
// [errors.AsType] or the sentinel with [ErrInvalidNull] instead.
func (e *InvalidNullError) Error() string {
	var b strings.Builder
	b.WriteString("cannot return null for non-null ")
	if e.ListElement {
		b.WriteString("element of ")
	}
	b.WriteString(e.position())
	if e.DeclaredType != "" {
		b.WriteString(" (")
		b.WriteString(e.DeclaredType)
		b.WriteByte(')')
	}
	b.WriteString(": ")
	b.WriteString(e.explanation())
	return b.String()
}

// position locates the null as precisely as the error allows: its schema
// coordinate, else the bare field name when the owning type is unknown, else
// the response path, which is at least where the client saw the null. Only a
// hand-built FieldContext reaches past the first — generated code always names
// both halves. Falling back costs spec conformance that a message does not owe
// anyone, and keeps a reader pointed somewhere.
func (e *InvalidNullError) position() string {
	if coord := e.SchemaCoordinate(); coord != "" {
		return "field " + coord
	}
	if e.Field != "" {
		return "field " + e.Field
	}
	if path := e.Path.String(); path != "" {
		return "response position " + path
	}
	return "position"
}

// explanation returns the trailing clause of the message. A marshaler that
// nulled a present value outranks the rest: the value arrived intact, so naming
// the resolver or struct field would misdirect.
func (e *InvalidNullError) explanation() string {
	switch {
	case e.Source == NullSourceMarshaler:
		return "the marshaler returned null for a non-nil value"
	case e.ListElement:
		return "the list element was nil"
	case e.Source == NullSourceRuntimeNonNull:
		return "the field was marked non-null at runtime and resolved to nil"
	case e.Source == NullSourceResolver:
		return "the resolver returned nil"
	case e.Source == NullSourceModelMethod:
		return "the model method returned nil"
	case e.Source == NullSourceModelField:
		return "the model struct field was nil and the field has no resolver"
	// NullSourceUnknown, and any value gqlgen does not itself construct.
	default:
		return "the value was nil"
	}
}

// AddInvalidNullError reports that the position being resolved produced a null the
// schema forbids, reading the source off the field context. It does nothing if
// this path already errored — that error is the cause, and the spec asks for
// one error per position.
//
// Its callers are invisible from here: the marshalers emitted by
// codegen/type.gotpl, and resolveField. They pass declaredType because those
// marshalers serve both field and list-element positions; empty derives it.
func AddInvalidNullError(ctx context.Context, declaredType string) {
	addInvalidNullError(ctx, declaredType, "")
}

// AddInvalidNullFromMarshaler reports a non-null violation from a marshaler that
// mapped a non-nil value to null, as MarshalTime does for the zero time. It is
// separate from [AddInvalidNullError] because the field context cannot tell the two
// apart, and naming a nil resolver or struct field here would misdirect.
func AddInvalidNullFromMarshaler(ctx context.Context, declaredType string) {
	addInvalidNullError(ctx, declaredType, NullSourceMarshaler)
}

// addInvalidNullError records the violation unless this path already errored. An
// empty source is derived from the field context.
func addInvalidNullError(ctx context.Context, declaredType string, source NullSource) {
	fc := GetFieldContext(ctx)
	if HasFieldError(ctx, fc) {
		return
	}
	AddError(ctx, newInvalidNullError(fc, declaredType, source))
}

func newInvalidNullError(
	fc *FieldContext,
	declaredType string,
	source NullSource,
) *InvalidNullError {
	err := &InvalidNullError{
		DeclaredType: declaredType,
		Source:       source,
		Path:         fc.Path(),
	}

	// Find the context naming the field. A list element pushes one carrying
	// only an index, so the name may be an ancestor's — and passing an index is
	// what marks this null as an element. Only an index counts: object.gotpl
	// pushes a context carrying just an Object for root types, and walking past
	// that says nothing about lists.
	var (
		named   *FieldContext
		name    string
		element bool
	)
	for named = fc; named != nil; named = named.Parent {
		name = fieldName(named)
		if name != "" {
			break
		}
		if named.Index != nil {
			element = true
		}
	}

	// Describe the position that broke its contract. Element-ness is known
	// whether or not anything named the field.
	err.ListElement = element
	if named != nil {
		err.Object = named.Object
		err.Field = name
		if err.DeclaredType == "" && named.Field.Definition != nil {
			err.DeclaredType = named.Field.Definition.Type.String()
		}
	}

	// An explicit source wins; otherwise read it off the field context.
	if err.Source == "" {
		if named == nil {
			err.Source = NullSourceUnknown
		} else {
			err.Source = nullSourceFor(named)
		}
	}
	return err
}

// nullSourceFor classifies which Go construct behind a field produced a nil.
// List elements are not its business; [InvalidNullError.ListElement] carries those.
func nullSourceFor(fc *FieldContext) NullSource {
	schemaNonNull := fc.Field.Definition != nil && fc.Field.Definition.Type.NonNull

	switch {
	case fc.NonNull && !schemaNonNull:
		// MarkNonNull raised a field the schema lets be null.
		return NullSourceRuntimeNonNull
	case fc.IsResolver:
		return NullSourceResolver
	case fc.IsMethod:
		// Generated code sets IsMethod for resolvers too; IsResolver ruled out above.
		return NullSourceModelMethod
	default:
		return NullSourceModelField
	}
}

// fieldName returns the field's schema name, or "" for a context naming no
// field — the one a list element pushes for its index. Falls back to the alias.
func fieldName(fc *FieldContext) string {
	if fc.Field.Field == nil {
		return ""
	}
	if fc.Field.Name != "" {
		return fc.Field.Name
	}
	return fc.Field.Alias
}
