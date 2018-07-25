package codegen

func (cfg *Config) buildDirectives() (directives []*Directive) {
	for name := range cfg.schema.Directives {
		if name == "skip" || name == "include" || name == "deprecated" {
			continue
		}
		directives = append(directives, &Directive{
			name: name,
		})
	}
	return directives
}
