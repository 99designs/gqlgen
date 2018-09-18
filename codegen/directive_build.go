package codegen

import (
	"go/types"
	"sort"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/loader"
)

func (cfg *Config) buildDirectives(types NamedTypes, imports *Imports, prog *loader.Program) (Directives, error) {
	var directives Directives

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

		directive := &Directive{
			Name: name,
			Args: args,
		}

		if impl := cfg.Directives.ImplementationFor(name); impl != "" {
			if err := bindDirectiveImplementation(directive, impl, imports, prog); err != nil {
				return nil, errors.Errorf("directive implementation for %s: %s", name, err.Error())
			}
		}

		directives = append(directives, directive)
	}

	sort.Slice(directives, func(i, j int) bool { return directives[i].Name < directives[j].Name })

	return directives, nil
}

func bindDirectiveImplementation(dir *Directive, impl string, imports *Imports, prog *loader.Program) error {
	pkg, goType := pkgAndType(impl)
	dir.Implementation = &Ref{
		Package: pkg,
		GoType:  goType,
		Import:  imports.add(pkg),
	}

	def, err := findGoType(prog, dir.Implementation.Package, goType)
	if err != nil {
		return err
	}
	if def == nil {
		return errors.Errorf("implementation %s not found", goType)
	}

	fn, ok := def.(*types.Func)
	if !ok {
		return errors.Errorf("implementation %s not a func", goType)
	}

	params := fn.Type().(*types.Signature).Params()
	if err := dir.validateParams(params); err != nil {
		return errors.Errorf("implementation %s should match signature %s: %s", goType, dir.Signature(), err.Error())
	}

	return nil
}
