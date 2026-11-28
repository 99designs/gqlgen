package tableimpl

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
)

// arg is one argument of a field or directive, as Schema.Link reads it from the schema.
type arg[EC Context] struct {
	name string
	// directives are the directives applied to the argument. They receive the map of
	// raw arguments.
	directives []directive[EC]
	// nilOK is set when a directive may return nil for the argument, which is when
	// the argument's Go type can be nil.
	nilOK bool
	// withNull runs the directives even when the argument is absent, as set by
	// call_argument_directives_with_null.
	withNull bool
	// typ unmarshals the argument.
	typ *In[EC]
	// goType is the Go type of the argument that errors name, when it is not the name of
	// typ. See Arg.GoType.
	goType string
}

// Arg describes how one argument of a field or directive is unmarshaled from Go.
// Generated code sets its name and what ties it to Go; Schema.Link finds the argument of
// that name in the schema and reads the rest from there, so that the tables keep working
// when the schema declares the arguments in another order, and Link panics when the
// schema has no argument of that name.
type Arg[EC Context] struct {
	Name string
	// In unmarshals the argument.
	In *In[EC]
	// Type is set instead of In when Link derives the In from the type of the argument
	// and its Go type, which Type gives as a nil pointer to it. An argument with neither
	// In nor Type is not bound to Go, such as one that the method of the field does not
	// take: it is left out, as the functions mode leaves it out of its arguments function.
	Type any
	// GoType is the Go type of the argument that errors name, as the generated package
	// writes it. Generated code gives it where the runtime would write the type
	// otherwise: reflect does not tell []byte from []uint8, nor interface{} from any.
	GoType string
}

// parseArgs unmarshals rawArgs into a map that holds a value of the argument's Go type
// for every argument, absent ones included. withPath adds the name of each argument to
// the path of its errors. The functions mode leaves it out for the arguments of the
// directives in the schema, which it unmarshals in place.
func parseArgs[EC Context](
	ctx context.Context,
	ec EC,
	args []arg[EC],
	rawArgs map[string]any,
	withPath bool,
) (map[string]any, error) {
	res := make(map[string]any, len(args))
	for i := range args {
		v, err := args[i].parse(ctx, ec, rawArgs, withPath)
		if err != nil {
			return nil, err
		}
		res[args[i].name] = v
	}
	return res, nil
}

func (a *arg[EC]) parse(
	ctx context.Context,
	ec EC,
	rawArgs map[string]any,
	withPath bool,
) (any, error) {
	raw, ok := rawArgs[a.name]
	if len(a.directives) == 0 {
		if !ok {
			return a.typ.zero, nil
		}
		if withPath {
			ctx = graphql.WithPathContext(ctx, graphql.NewPathWithField(a.name))
		}
		return a.typ.unmarshal(ctx, ec, raw)
	}

	if !ok && !a.withNull {
		return a.typ.zero, nil
	}
	ctx = graphql.WithPathContext(ctx, graphql.NewPathWithField(a.name))
	next := func(ctx context.Context) (any, error) {
		// The directives receive rawArgs and may change it, so the argument is read when
		// they call next, as in the functions mode.
		raw, ok := rawArgs[a.name]
		if !ok {
			return a.typ.zero, nil
		}
		return a.typ.unmarshal(ctx, ec, raw)
	}
	tmp, err := chain(ec, rawArgs, a.directives, next, failure{zero: a.typ.zero})(ctx)
	if err != nil {
		return a.typ.zero, graphql.ErrorOnPath(ctx, err)
	}
	if a.typ.accept(tmp) {
		return tmp, nil
	}
	if a.nilOK && tmp == nil {
		return a.typ.zero, nil
	}
	goType := a.typ.name
	if a.goType != "" {
		goType = a.goType
	}
	return a.typ.zero, graphql.ErrorOnPath(ctx, fmt.Errorf(
		"unexpected type %T from directive, should be %s", tmp, goType))
}
