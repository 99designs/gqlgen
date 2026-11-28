package codegen

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
)

type DirectiveList map[string]*Directive

// LocationDirectives filter directives by location
func (dl DirectiveList) LocationDirectives(location string) DirectiveList {
	return locationDirectives(dl, ast.DirectiveLocation(location))
}

type Directive struct {
	*ast.DirectiveDefinition
	Name string
	Args []*FieldArgument

	config.DirectiveConfig
}

// IsLocation check location directive
func (d *Directive) IsLocation(location ...ast.DirectiveLocation) bool {
	for _, l := range d.Locations {
		if slices.Contains(location, l) {
			return true
		}
	}

	return false
}

func locationDirectives(
	directives DirectiveList,
	location ...ast.DirectiveLocation,
) map[string]*Directive {
	mDirectives := make(map[string]*Directive)
	for name, d := range directives {
		if d.IsLocation(location...) {
			mDirectives[name] = d
		}
	}
	return mDirectives
}

// directiveArgVarName returns the name of the variable that the generated code unmarshals
// the i-th argument of a directive into. The variable sits in a closure beside the object
// that the directive receives, the next resolver, the receiver of the generated code, the
// packages it calls and the literals it writes for the other arguments, so it is named by
// position rather than after the argument, whose name could hide any of them.
func directiveArgVarName(i int) string {
	return fmt.Sprintf("arg%d", i)
}

// directiveParamNames returns the names of the parameters that the function of a directive
// in DirectiveRoot takes for the arguments args, by argument. They follow the arguments, but
// one that would repeat ctx, obj, next, the results res and err, or an earlier parameter gets
// the suffix Arg until it does not.
func directiveParamNames(args []*FieldArgument) []string {
	taken := map[string]bool{"ctx": true, "obj": true, "next": true, "res": true, "err": true}
	names := make([]string, len(args))
	for i, arg := range args {
		v := templates.ToGoPrivate(arg.Name)
		if v != "_" {
			for taken[v] {
				v += "Arg"
			}
			taken[v] = true
		}
		names[i] = v
	}
	return names
}

func (b *builder) buildDirectives() (map[string]*Directive, error) {
	directives := make(map[string]*Directive, len(b.Schema.Directives))

	// Name order keeps the bound argument references, and so the generated code,
	// stable between runs.
	for _, name := range slices.Sorted(maps.Keys(b.Schema.Directives)) {
		dir := b.Schema.Directives[name]
		if _, ok := directives[name]; ok {
			return nil, fmt.Errorf("directive with name %s already exists", name)
		}

		var args []*FieldArgument
		for i, arg := range dir.Arguments {
			tr, err := b.Binder.TypeReference(arg.Type, nil)
			if err != nil {
				return nil, err
			}

			newArg := &FieldArgument{
				ArgumentDefinition: arg,
				TypeReference:      tr,
				VarName:            directiveArgVarName(i),
			}

			if arg.DefaultValue != nil {
				var err error
				newArg.Default, err = arg.DefaultValue.Value(nil)
				if err != nil {
					return nil, fmt.Errorf(
						"default value for directive argument %s(%s) is not valid: %w",
						dir.Name,
						arg.Name,
						err,
					)
				}
			}
			args = append(args, newArg)
		}

		directives[name] = &Directive{
			DirectiveDefinition: dir,
			Name:                name,
			Args:                args,
			DirectiveConfig:     b.Config.Directives[name],
		}
	}

	return directives, nil
}

func (b *builder) getDirectives(list ast.DirectiveList) ([]*Directive, error) {
	dirs := make([]*Directive, len(list))
	for i, d := range list {
		argValues := make(map[string]any, len(d.Arguments))
		for _, da := range d.Arguments {
			val, err := da.Value.Value(nil)
			if err != nil {
				return nil, err
			}
			argValues[da.Name] = val
		}
		def, ok := b.Directives[d.Name]
		if !ok {
			return nil, fmt.Errorf("directive %s not found", d.Name)
		}

		var args []*FieldArgument
		for _, a := range def.Args {
			value := a.Default
			if argValue, ok := argValues[a.Name]; ok {
				value = argValue
			}
			args = append(args, &FieldArgument{
				ArgumentDefinition: a.ArgumentDefinition,
				Value:              value,
				VarName:            a.VarName,
				TypeReference:      a.TypeReference,
			})
		}
		dirs[i] = &Directive{
			Name:                d.Name,
			Args:                args,
			DirectiveDefinition: list[i].Definition,
			DirectiveConfig:     b.Config.Directives[d.Name],
		}
	}

	return dirs, nil
}

func (d *Directive) ArgsFunc() string {
	if len(d.Args) == 0 {
		return ""
	}

	return "dir_" + d.Name + "_args"
}

func (d *Directive) CallArgs() string {
	args := []string{"ctx", "obj", "n"}

	for _, arg := range d.Args {
		args = append(
			args,
			fmt.Sprintf(
				"args[%q].(%s)",
				arg.Name,
				templates.CurrentImports.LookupType(arg.TypeReference.GO),
			),
		)
	}

	return strings.Join(args, ", ")
}

func (d *Directive) ResolveArgs(obj string, next int) string {
	args := []string{"ctx", obj, fmt.Sprintf("directive%d", next)}

	for _, arg := range d.Args {
		dArg := arg.VarName
		if arg.Value == nil && arg.Default == nil {
			dArg = "nil"
		}

		args = append(args, dArg)
	}

	return strings.Join(args, ", ")
}

func (d *Directive) CallName() string {
	return ucFirst(d.Name)
}

func (d *Directive) Declaration() string {
	res := d.CallName() + " func(ctx context.Context, obj any, next graphql.Resolver"

	var resSb173 strings.Builder
	names := directiveParamNames(d.Args)
	for i, arg := range d.Args {
		fmt.Fprintf(
			&resSb173,
			", %s %s",
			names[i],
			templates.CurrentImports.LookupType(arg.TypeReference.GO),
		)
	}
	res += resSb173.String()

	res += ") (res any, err error)"
	return res
}

func (d *Directive) IsBuiltIn() bool {
	return d.Implementation != nil
}

func (d *Directive) CallPath() string {
	if d.IsBuiltIn() {
		return "builtInDirective" + d.CallName()
	}

	return "ec.Directives." + d.CallName()
}

func (d *Directive) FunctionImpl() string {
	if d.Implementation == nil {
		return ""
	}

	return d.CallPath() + " = " + *d.Implementation
}
