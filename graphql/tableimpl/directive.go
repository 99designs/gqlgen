package tableimpl

import (
	"context"
	"errors"
	"slices"
	"strconv"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/graphql"
)

// DirectiveDef is the implementation of a directive declared in the schema. Generated
// code declares one per directive that the tables run, with Init.
type DirectiveDef[EC Context] struct {
	Name string
	// Args unmarshal the arguments of the directive, every one that the schema declares,
	// in the order the schema declares them. Link matches them with the schema by name.
	Args []Arg[EC]
	// Call calls the directive's implementation with the unmarshaled arguments.
	Call func(ctx context.Context, ec EC, obj any, next graphql.Resolver, args map[string]any) (any, error)

	// implemented reports whether the configuration gives the directive an
	// implementation. It is nil for the directives that gqlgen implements itself.
	implemented func(ec EC) bool
	locations   []ast.DirectiveLocation
	args        []arg[EC]
}

// Init sets the directive to name, args and call, and registers it with s. implemented
// reports whether the configuration of a request gives the directive an implementation,
// or is nil for a directive that always has one: a directive without one fails with the
// error of the functions mode rather than call call.
//
//go:noinline
func (d *DirectiveDef[EC]) Init(
	s *Schema[EC],
	name string,
	args []Arg[EC],
	call func(ctx context.Context, ec EC, obj any, next graphql.Resolver, args map[string]any) (any, error),
	implemented func(ec EC) bool,
) {
	*d = DirectiveDef[EC]{Name: name, Args: args, Call: call, implemented: implemented}
	if _, ok := s.directives[name]; ok {
		panic("tableimpl:directive " + strconv.Quote(name) + " registered twice")
	}
	s.directives[name] = d
}

func (d *DirectiveDef[EC]) link(s *Schema[EC], name string) {
	decl := s.schema.Directives[name]
	if decl == nil {
		panic("tableimpl:the schema declares no directive " + strconv.Quote(name))
	}
	d.locations = decl.Locations
	d.args = s.args(decl.Arguments, d.Args, "directive @"+name)
	// The functions mode unmarshals the arguments of a directive in place, without the
	// directives that the schema puts on the arguments of its definition.
	for i := range d.args {
		d.args[i].directives = nil
	}
}

// declares reports whether the directive may be applied at one of the locations.
func (d *DirectiveDef[EC]) declares(locations ...ast.DirectiveLocation) bool {
	return slices.ContainsFunc(d.locations, func(l ast.DirectiveLocation) bool {
		return slices.Contains(locations, l)
	})
}

// directive is a directive applied in the schema, with the argument values written
// there.
type directive[EC Context] struct {
	def *DirectiveDef[EC]
	// args holds the argument values from the schema, or their defaults. Arguments
	// without either, or with null, are absent.
	args map[string]any
}

// failure is what a directive returns when it cannot call its implementation, because
// its arguments fail to unmarshal or it has none, as in the functions mode: zero, the zero
// value of the Go type of the element the directive is applied to, and the error, on the
// path of the context when onPath is set.
type failure struct {
	zero   any
	onPath bool
}

func (f failure) of(ctx context.Context, err error) (any, error) {
	if f.onPath {
		return f.zero, graphql.ErrorOnPath(ctx, err)
	}
	return f.zero, err
}

// chain wraps next with dirs. The first directive is called last, right before next,
// as the directives written first on a schema element are the innermost.
func chain[EC Context](
	ec EC,
	obj any,
	dirs []directive[EC],
	next graphql.Resolver,
	failed failure,
) graphql.Resolver {
	for i := range dirs {
		next = dirs[i].wrap(ec, obj, next, failed)
	}
	return next
}

func (d *directive[EC]) wrap(
	ec EC,
	obj any,
	next graphql.Resolver,
	failed failure,
) graphql.Resolver {
	return func(ctx context.Context) (any, error) {
		var args map[string]any
		if len(d.def.args) > 0 {
			var err error
			raw, _ := cloneValue(d.args).(map[string]any)
			if args, err = parseArgs(ctx, ec, d.def.args, raw, false); err != nil {
				return failed.of(ctx, err)
			}
		}
		if d.def.implemented != nil && !d.def.implemented(ec) {
			return failed.of(ctx, errors.New("directive "+d.def.Name+" is not implemented"))
		}
		return d.def.Call(ctx, ec, obj, next, args)
	}
}
