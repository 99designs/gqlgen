package codegen

import (
	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/ast"
)

func (cfg *Config) buildDirectives(types NamedTypes) (map[string]*Directive, error) {
	directives := make(map[string]*Directive, len(cfg.schema.Directives))

	for name, dir := range cfg.schema.Directives {
		if _, ok := directives[name]; ok {
			return nil, errors.Errorf("directive with name %s already exists", name)
		}
		if name == "skip" || name == "include" || name == "deprecated" {
			continue
		}

		var args []FieldArgument
		for _, arg := range dir.Arguments {
			newArg := FieldArgument{
				GQLName:   arg.Name,
				Type:      types.getType(arg.Type),
				GoVarName: sanitizeArgName(arg.Name),
			}

			if !newArg.Type.IsInput && !newArg.Type.IsScalar {
				return nil, errors.Errorf("%s cannot be used as argument of directive %s(%s) only input and scalar types are allowed", arg.Type, dir.Name, arg.Name)
			}

			if arg.DefaultValue != nil {
				var err error
				newArg.Default, err = arg.DefaultValue.Value(nil)
				if err != nil {
					return nil, errors.Errorf("default value for directive argument %s(%s) is not valid: %s", dir.Name, arg.Name, err.Error())
				}
			}
			args = append(args, newArg)
		}

		directives[name] = &Directive{
			Name: name,
			Args: args,
		}
	}

	return directives, nil
}

func (cfg *Config) getDirectives(list ast.DirectiveList) ([]*Directive, error) {

	dirs := make([]*Directive, len(list))
	for i, d := range list {
		argValues := make(map[string]interface{}, len(d.Arguments))
		for _, da := range d.Arguments {
			val, err := da.Value.Value(nil)
			if err != nil {
				return nil, err
			}
			argValues[da.Name] = val
		}

		if def, ok := cfg.Directives[d.Name]; ok {
			var args []FieldArgument
			for _, a := range def.Args {

				value := a.Default
				if argValue, ok := argValues[a.GQLName]; ok {
					value = argValue
				}
				args = append(args, FieldArgument{
					GQLName:   a.GQLName,
					Value:     value,
					GoVarName: a.GoVarName,
					Type:      a.Type,
				})
			}
			dirs[i] = &Directive{
				Name: d.Name,
				Args: args,
			}
		}
	}

	return dirs, nil
}
