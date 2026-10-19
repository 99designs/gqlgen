package tableimpl

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync/atomic"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/graphql"
)

// Marshal converts V into a response value. Generated code passes its marshal
// functions.
type Marshal[EC, V any] func(ctx context.Context, ec EC, sel ast.SelectionSet, v V) graphql.Marshaler

// Resolve resolves the field at index field of the table of an object type on obj, with
// the arguments unmarshaled into the field context. It returns any, so that directives
// and middleware can replace the value. Generated code declares one per object type, a
// function that switches on the index: what the compiler spends on a package grows with
// the number of its functions, and a closure per field would add one for every field of
// the schema. A field read from its object declares the value with the Go type that the
// field's Out marshals before it returns it, so that the compiler checks that the field
// has that type.
type Resolve[EC any] func(ctx context.Context, ec EC, obj any, field int, args map[string]any) (any, error)

// NoField is the error of a Resolve for an index that is not a field of its table, which
// means that the code was generated wrong.
func NoField(field int) error {
	return fmt.Errorf("tableimpl:the table has no field %d", field)
}

// Context is implemented by the generated executionContext: table mode generates OpCtx and
// AddDeferred, and ProcessDeferredGroup comes from the graphql.ExecutionContextState it
// embeds.
type Context interface {
	OpCtx() *graphql.OperationContext
	AddDeferred(n int32)
	ProcessDeferredGroup(dg graphql.DeferredGroup)
}

// fieldMiddleware is implemented by the generated executionContext when the schema
// declares directives that queries may put on fields. FieldMiddleware applies them.
type fieldMiddleware interface {
	FieldMiddleware(ctx context.Context, obj any, next graphql.Resolver) graphql.Resolver
}

// Object describes how a GraphQL object type is resolved from its Go value.
//
// Generated code declares one Object per object type in the tables of the executable
// schema, and calls Init with the Resolve of the type and its fields. What the schema
// says about the type, such as
// its interfaces and the types of its fields, Schema.Link reads from the schema. The
// object is passed around as any, so that one instantiation of the runtime serves every
// object type; the root operation types have no value and pass nil.
type Object[EC Context] struct {
	Name string

	implementors []string
	// root is set for the query and mutation types. Their fields run inside the root
	// field middleware. The subscription type is resolved by Subscribe instead of
	// Marshal.
	root bool
	// omitPanicHandler is set when the generated code was configured not to recover
	// panics in resolvers.
	omitPanicHandler bool
	// directives are the directives the schema declares on the object type. They run
	// around the resolver of every field that returns the type, inside the field's
	// own directives, as in the functions mode.
	directives []directive[EC]

	// resolver resolves the fields of the object by their index.
	resolver Resolve[EC]
	fields   []Field[EC]
	index    map[string]*Field[EC]
}

// Field describes how one field of an object is resolved from Go. Generated code sets
// its name and what ties it to Go; Schema.Link reads the rest from the schema.
type Field[EC Context] struct {
	Name string
	// Concurrent is set for fields that are resolved in their own goroutine, and may
	// be deferred, besides the resolvers, which Link marks itself.
	Concurrent bool
	// IsMethod is set for fields read by calling a method, besides the resolvers.
	IsMethod   bool
	IsResolver bool
	// Args unmarshal the field's arguments, every one that the schema declares, in the
	// order the functions mode unmarshals them: the order the schema declares them, or
	// for a field bound to a method, the order of its parameters. Link matches them with
	// the schema by name.
	Args []Arg[EC]
	// Out marshals the value of the field's Go type. A value of another type that
	// implements graphql.Marshaler is used as it is.
	Out *Out[EC]
	// Type is set instead of Out when Link derives the Out from the type of the field
	// and its Go type, which Type gives as a nil pointer to it.
	Type any
	// Stream is set for the fields of the subscription type. It receives the events
	// from the channel the resolver returns, and Out marshals each of them.
	Stream *Stream
	// ResolveBatch is set for the fields that a batch resolver resolves, in place of the
	// Resolve of the object. It takes the field as the request selects it, by whose
	// response name the batch resolver shares its results between the parents, and reads
	// the arguments from the field context, as in the functions mode.
	ResolveBatch func(ctx context.Context, ec EC, obj any, field graphql.CollectedField) (any, error)

	// index is the position of the field in the table, by which the Resolve of the object
	// resolves it.
	index int
	// nonNull is set when the schema declares the field non-null.
	nonNull bool
	// rootValue is set for fields that return a root operation type, which are
	// marshaled without middleware or null checks.
	rootValue bool

	// Exactly one of children, leaf and abstract describes the field's type. They decide
	// what FieldContext.Child does.
	//
	// children is the object the field returns.
	children *Object[EC]
	// leaf is the name of a type without fields, such as a scalar, enum or union.
	leaf string
	// abstract is the kind of a type with fields that is not an object, such as INTERFACE.
	abstract string

	args []arg[EC]
	// directives are the directives the schema declares on the field, after those of
	// its type when the type is not an object. Those of an object type the field
	// returns are on children. The FIELD directives of the query are applied around
	// them by the executionContext's FieldMiddleware.
	directives []directive[EC]
}

// Out marshals the values of one Go type, given as any. Generated code declares one per
// type that fields return, with OutOf.
type Out[EC any] struct {
	accept  func(v any) bool
	marshal func(ctx context.Context, ec EC, sel ast.SelectionSet, v any) graphql.Marshaler
	name    string
	// zero is the zero value of the Go type, which a directive that cannot run returns.
	zero any
}

// OutFor is the Out of the values of type V, with the Marshal of the type, which the
// generated code calls where it marshals values of the type itself: for the elements of
// lists, and in the functions it generates for the type references the combinators do not
// cover.
type OutFor[EC, V any] struct {
	*Out[EC]
	Marshal Marshal[EC, V]
}

// OutOf returns the Out of the values that m marshals.
//
//go:noinline
func OutOf[EC, V any](m Marshal[EC, V]) OutFor[EC, V] {
	var zero V
	return OutFor[EC, V]{Out: &Out[EC]{
		accept: func(v any) bool {
			_, ok := v.(V)
			return ok
		},
		marshal: func(ctx context.Context, ec EC, sel ast.SelectionSet, v any) graphql.Marshaler {
			// A nil of an interface type V reaches here as a nil any, from a
			// subscription event.
			x, _ := v.(V)
			return m(ctx, ec, sel, x)
		},
		name: fmt.Sprintf("%T", zero),
		zero: zero,
	}, Marshal: m}
}

// Init sets the object to the type called name, whose fields resolve resolves by their
// index in fields, and registers it with s. Generated code calls it once, when it builds
// the tables of an executable schema.
//
//go:noinline
func (o *Object[EC]) Init(s *Schema[EC], name string, resolve Resolve[EC], fields []Field[EC]) {
	*o = Object[EC]{
		Name:     name,
		resolver: resolve,
		fields:   fields,
		index:    make(map[string]*Field[EC], len(fields)),
	}
	for i := range fields {
		fields[i].index = i
		o.index[fields[i].Name] = &o.fields[i]
	}
	if _, ok := s.objects[name]; ok {
		panic("tableimpl:object " + strconv.Quote(name) + " registered twice")
	}
	s.objects[name] = o
}

// link reads what the tables leave to the schema: see Schema.Link.
func (o *Object[EC]) link(s *Schema[EC], name string) {
	def := s.definition(name)
	o.root = s.isRoot(def)
	o.omitPanicHandler = s.OmitPanicHandler
	o.implementors = []string{name}
	for _, abstract := range s.schema.GetImplements(def) {
		o.implementors = append(o.implementors, abstract.Name)
	}
	o.directives = s.applied(def.Directives, ast.LocationFieldDefinition)
	// The fields of the mutation type run one after the other.
	sequential := def == s.schema.Mutation
	for i := range o.fields {
		f := &o.fields[i]
		fd := def.Fields.ForName(f.Name)
		if fd == nil {
			panic(fmt.Sprintf("tableimpl:type %s has no field %s", name, f.Name))
		}
		f.link(s, fd, sequential)
	}
}

func (f *Field[EC]) link(s *Schema[EC], fd *ast.FieldDefinition, sequential bool) {
	f.nonNull = fd.Type.NonNull
	f.IsMethod = f.IsMethod || f.IsResolver
	f.Concurrent = f.Concurrent || (f.IsResolver && !sequential)
	typ := s.definition(fd.Type.Name())
	f.rootValue = s.isRoot(typ)
	switch {
	case typ.Kind == ast.Object:
		f.children = s.objects[typ.Name]
		if f.children == nil {
			panic("tableimpl:the tables have no object " + strconv.Quote(typ.Name))
		}
	case len(typ.Fields) > 0:
		f.abstract = string(typ.Kind)
	default:
		f.leaf = typ.Name
	}
	// The directives of a type that is not an object run around the fields that
	// return it, before the field's own; those of an object type run through children.
	if typ.Kind != ast.Object {
		f.directives = s.applied(typ.Directives, ast.LocationFieldDefinition)
	}
	f.directives = append(f.directives, s.applied(fd.Directives, "")...)
	f.args = s.args(fd.Arguments, f.Args, "field "+fd.Name)
	if f.Out == nil && f.Type != nil {
		t, ok := goType(f.Type)
		if !ok {
			panic(notGoType("field "+fd.Name, f.Type))
		}
		f.Out = s.derivedOut(fd.Type, t)
	}
}

// ParseArgs unmarshals the raw arguments of the field called name. The generated
// complexity function uses it.
func (o *Object[EC]) ParseArgs(
	ctx context.Context,
	ec EC,
	name string,
	rawArgs map[string]any,
) (map[string]any, error) {
	f := o.index[name]
	if f == nil {
		return nil, fmt.Errorf("no field named %q was found under type %s", name, o.Name)
	}
	return parseArgs(ctx, ec, f.args, rawArgs, true)
}

// childFieldContext returns the field context of field, selected under this object. It
// backs FieldContext.Child for fields that return this object.
func (o *Object[EC]) childFieldContext(
	ctx context.Context,
	ec EC,
	field graphql.CollectedField,
) (*graphql.FieldContext, error) {
	f := o.index[field.Name]
	if f == nil {
		return nil, fmt.Errorf("no field named %q was found under type %s", field.Name, o.Name)
	}
	return o.fieldContext(ctx, ec, f, field)
}

// Marshal resolves the fields of obj selected by sel.
func (o *Object[EC]) Marshal(
	ctx context.Context,
	ec EC,
	sel ast.SelectionSet,
	obj any,
) graphql.Marshaler {
	oc := ec.OpCtx()
	fields := graphql.CollectFields(oc, sel, o.implementors)
	if o.root {
		ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{Object: o.Name})
	}

	out := graphql.NewFieldSet(fields)
	var deferred graphql.DeferredGroup
	for i, field := range fields {
		if field.Name == "__typename" {
			out.Values[i] = graphql.MarshalString(o.Name)
			continue
		}
		f := o.index[field.Name]
		if f == nil {
			panic("unknown field " + strconv.Quote(field.Name))
		}

		var rootCtx context.Context
		if o.root {
			rootCtx = graphql.WithRootFieldContext(ctx, &graphql.RootFieldContext{
				Object: field.Name,
				Field:  field,
			})
		}

		if !f.Concurrent {
			if o.root {
				out.Values[i] = oc.RootResolverMiddleware(
					rootCtx,
					func(ctx context.Context) graphql.Marshaler {
						return o.resolveField(ctx, ec, f, field, obj)
					},
				)
			} else {
				out.Values[i] = o.resolveField(ctx, ec, f, field, obj)
			}
			if f.isInvalid(out.Values[i]) {
				atomic.AddUint32(&out.Invalids, 1)
			}
			continue
		}

		// The concurrent fields are scheduled as the functions mode schedules them.
		resolve := func(ctx context.Context) graphql.Marshaler {
			return o.resolveField(ctx, ec, f, field, obj)
		}
		if o.root {
			out.ResolveRootConcurrently(rootCtx, oc, i, !o.omitPanicHandler, f.nonNull, resolve)
			continue
		}
		// The functions mode makes the group for every object; it is only needed for a
		// deferred field, so it is made for the first.
		if field.IsDeferred() && deferred.FieldSet == nil {
			deferred = graphql.NewDeferredGroup(ctx)
		}
		out.ResolveConcurrently(oc, &deferred, i, !o.omitPanicHandler, f.nonNull, resolve)
	}

	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	if n := len(deferred.Defers); n > 0 {
		ec.AddDeferred(int32(min(n, math.MaxInt32)))
		ec.ProcessDeferredGroup(deferred)
	}

	return out
}

// isInvalid reports whether a resolved value makes the object null: a null in a
// non-null field, or a null that a field marked non-null at runtime asks to propagate.
func (f *Field[EC]) isInvalid(res graphql.Marshaler) bool {
	if f.nonNull {
		return res == graphql.Null
	}
	return res == graphql.RequiredNull
}

// resolveField resolves one field of obj, as graphql.ResolveField does for the
// functions mode.
func (o *Object[EC]) resolveField(
	ctx context.Context,
	ec EC,
	f *Field[EC],
	field graphql.CollectedField,
	obj any,
) (ret graphql.Marshaler) {
	oc := ec.OpCtx()
	fc, err := o.fieldContext(ctx, ec, f, field)
	if err != nil {
		return graphql.Null
	}
	ctx = graphql.WithFieldContext(ctx, fc)

	if !o.omitPanicHandler {
		defer func() {
			if r := recover(); r != nil {
				oc.Error(ctx, oc.Recover(ctx, r))
				ret = graphql.Null
			}
		}()
	}

	if f.rootValue {
		// The field returns an empty value of a root operation type, which cannot fail.
		res, _ := o.resolver(ctx, ec, obj, f.index, nil)
		return f.marshal(ctx, ec, field.Selections, fc, res)
	}

	res, err := o.resolve(&ctx, ec, f, field, obj)
	if err != nil {
		oc.Error(ctx, graphql.AddFieldLocationToError(ctx, err))
		if fc.NonNull && !f.nonNull {
			return graphql.RequiredNull
		}
		return graphql.Null
	}
	if res == nil {
		if f.nonNull || fc.NonNull {
			graphql.AddInvalidNullError(ctx, "")
		}
		if fc.NonNull && !f.nonNull {
			return graphql.RequiredNull
		}
		return graphql.Null
	}
	return f.marshal(ctx, ec, field.Selections, fc, res)
}

// resolve runs the resolver of the field f of o on obj inside the field's directives, the
// FIELD directives of the query and the resolver middleware. It sets *ctx to the context
// that the middleware passed to the resolver, which the children of the field use, and
// which the recover function gets when the resolver panics, as in the functions mode.
func (o *Object[EC]) resolve(
	ctx *context.Context,
	ec EC,
	f *Field[EC],
	field graphql.CollectedField,
	obj any,
) (any, error) {
	var next graphql.Resolver
	if f.ResolveBatch != nil {
		// Only this closure holds the field, so that the others stay as small.
		next = func(rctx context.Context) (any, error) {
			*ctx = rctx
			return f.ResolveBatch(rctx, ec, obj, field)
		}
	} else {
		next = func(rctx context.Context) (any, error) {
			*ctx = rctx // use context from middleware stack in children
			var args map[string]any
			// Only the resolvers and the methods take arguments, whose values the
			// functions mode reads from the field context there. Middleware may have
			// replaced it.
			if len(f.args) > 0 && (f.IsResolver || f.IsMethod) {
				args = graphql.GetFieldContext(rctx).Args
			}
			return o.resolver(rctx, ec, obj, f.index, args)
		}
	}
	failed := failure{zero: f.Out.zero}
	if f.children != nil {
		next = chain(ec, obj, f.children.directives, next, failed)
	}
	next = chain(ec, obj, f.directives, next, failed)
	if fm, ok := any(ec).(fieldMiddleware); ok {
		next = fm.FieldMiddleware(*ctx, obj, next)
	}
	return ec.OpCtx().ResolverMiddleware(*ctx, next)
}

// marshal marshals the resolved value res of f.
func (f *Field[EC]) marshal(
	ctx context.Context,
	ec EC,
	sel ast.SelectionSet,
	fc *graphql.FieldContext,
	res any,
) graphql.Marshaler {
	if f.Out.accept(res) {
		fc.Result = res
		return f.Out.marshal(ctx, ec, sel, res)
	}
	if m, ok := res.(graphql.Marshaler); ok {
		fc.Result = m
		return m
	}
	graphql.AddErrorf(
		ctx,
		`unexpected type %T from middleware/directive chain, should be %s`,
		res,
		f.Out.name,
	)
	return graphql.Null
}

// fieldContext builds the field context of f and unmarshals its arguments.
func (o *Object[EC]) fieldContext(
	ctx context.Context,
	ec EC,
	f *Field[EC],
	field graphql.CollectedField,
) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     o.Name,
		Field:      field,
		IsMethod:   f.IsMethod,
		IsResolver: f.IsResolver,
		Child:      f.child(ec),
	}
	if len(f.args) == 0 {
		return fc, nil
	}

	oc := ec.OpCtx()
	if !o.omitPanicHandler {
		defer func() {
			if r := recover(); r != nil {
				err = oc.Recover(ctx, r)
				oc.Error(ctx, err)
			}
		}()
	}
	ctx = graphql.WithFieldContext(ctx, fc)
	rawArgs := field.ArgumentMap(oc.Variables)
	if fc.Args, err = parseArgs(ctx, ec, f.args, rawArgs, true); err != nil {
		oc.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

// child returns the FieldContext.Child function of f.
func (f *Field[EC]) child(
	ec EC,
) func(context.Context, graphql.CollectedField) (*graphql.FieldContext, error) {
	switch {
	case f.children != nil:
		children := f.children
		return func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return children.childFieldContext(ctx, ec, field)
		}
	case f.abstract != "":
		kind := f.abstract
		return func(context.Context, graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("FieldContext.Child cannot be called on type " + kind)
		}
	default:
		leaf := f.leaf
		return func(context.Context, graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type " + leaf + " does not have child fields")
		}
	}
}
