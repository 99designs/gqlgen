package ast

func arg2map(defs ArgumentDefinitionList, args ArgumentList, vars map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{}
	var err error

	for _, argDef := range defs {
		argValue := args.ForName(argDef.Name)
		if argValue == nil {
			if argDef.DefaultValue != nil {
				result[argDef.Name], err = argDef.DefaultValue.Value(vars)
				if err != nil {
					panic(err)
				}
			}
			continue
		}
		if argValue.Value.Kind == Variable {
			if val, ok := vars[argValue.Value.Raw]; ok {
				result[argDef.Name] = val
			}
			continue
		}
		result[argDef.Name], err = argValue.Value.Value(vars)
		if err != nil {
			panic(err)
		}
	}

	return result
}
