package codegen

import "github.com/vektah/gqlparser/v2/ast"

const concurrentDirectiveName = "concurrent"

func makeConcurrentObjectAndField(obj *Object, f *Field) {
	var hasConcurrentDirective bool
	for _, dir := range obj.Directives {
		if dir.Name == concurrentDirectiveName {
			hasConcurrentDirective = true
			break
		}
	}

	if !hasConcurrentDirective {
		obj.Directives = append(obj.Directives, &Directive{
			DirectiveDefinition: &ast.DirectiveDefinition{
				Name: concurrentDirectiveName,
			},
			Name:    concurrentDirectiveName,
			Builtin: true,
		})
		obj.DisableConcurrency = false
	}

	if obj.Definition != nil && obj.Definition.Directives.ForName(concurrentDirectiveName) == nil {
		obj.Definition.Directives = append(obj.Definition.Directives, &ast.Directive{
			Name: concurrentDirectiveName,
			Definition: &ast.DirectiveDefinition{
				Name: concurrentDirectiveName,
			},
		})
	}

	if f.TypeReference != nil && f.TypeReference.Definition != nil {
		for _, dir := range f.TypeReference.Definition.Directives {
			if dir.Name == concurrentDirectiveName {
				hasConcurrentDirective = true
				break
			}
		}

		if !hasConcurrentDirective {
			f.TypeReference.Definition.Directives = append(f.TypeReference.Definition.Directives, &ast.Directive{
				Name: concurrentDirectiveName,
			})
		}
	}
}
