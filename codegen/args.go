package codegen

type ArgSet struct {
	Args     []*FieldArgument
	FuncDecl string
}

func (a *Data) Args() map[string][]*FieldArgument {
	ret := map[string][]*FieldArgument{}
	for _, o := range a.Objects {
		for _, f := range o.Fields {
			if len(f.Args) > 0 {
				ret[f.ArgsFunc()] = f.Args
			}
		}
	}

	for _, d := range a.Directives {
		if len(d.Args) > 0 {
			ret[d.ArgsFunc()] = d.Args
		}
	}
	return ret
}
