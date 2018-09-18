package codegen

import (
	"go/types"
	"sort"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/loader"
)

func (cfg *Config) buildDirectives(types NamedTypes, imports *Imports, prog *loader.Program) ([]*Directive, error) {
	var directives []*Directive

	for name, dir := range cfg.schema.Directives {
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
				newArg.StripPtr()
			}
			args = append(args, newArg)
		}

		var impl *Ref
		if cfg.Directives.Exists(name) && cfg.Directives[name].Implementation != "" {
			impl = &Ref{}
			impl.Package, impl.GoType = pkgAndType(cfg.Directives[name].Implementation)
			impl.Import = imports.add(impl.Package)
			if err := bindDirectiveImplementation(prog, args, impl); err != nil {
				return nil, errors.Errorf("directive implementation for \"%s\": %s", name, err.Error())
			}
		}

		directives = append(directives, &Directive{
			Name:           name,
			Args:           args,
			Implementation: impl,
		})
	}

	sort.Slice(directives, func(i, j int) bool { return directives[i].Name < directives[j].Name })

	return directives, nil
}

func bindDirectiveImplementation(prog *loader.Program, args []FieldArgument, impl *Ref) error {
	def, err := findGoType(prog, impl.Package, impl.GoType)
	if err != nil {
		return err
	}
	if def == nil {
		return errors.Errorf("implementation not found")
	}

	_, ok := def.(*types.Func)
	if !ok {
		return errors.Errorf("implementation not a func")
	}

	// TODO match field args to implementation signature

	return nil
}
