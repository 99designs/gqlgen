package codegen

import (
	"sort"

	"go/types"

	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/99designs/gqlgen/codegen/unified"
)

type ModelBuild struct {
	Interfaces  []*unified.Interface
	Models      unified.Objects
	Enums       []unified.Enum
	PackageName string
}

// Create a list of models that need to be generated
func buildModels(s *unified.Schema) error {
	b := &ModelBuild{
		PackageName: s.Config.Model.Package,
	}

	for _, o := range s.Interfaces {
		if !o.InTypemap {
			b.Interfaces = append(b.Interfaces, o)
		}
	}

	for _, o := range append(s.Objects, s.Inputs...) {
		if !o.InTypemap {
			b.Models = append(b.Models, o)
		}
	}

	for _, e := range s.Enums {
		if !e.InTypemap {
			b.Enums = append(b.Enums, e)
		}
	}

	if len(b.Models) == 0 && len(b.Enums) == 0 {
		return nil
	}

	sort.Slice(b.Models, func(i, j int) bool {
		return b.Models[i].Definition.GQLDefinition.Name < b.Models[j].Definition.GQLDefinition.Name
	})

	err := templates.RenderToFile("models.gotpl", s.Config.Model.Filename, b)

	for _, model := range b.Models {
		modelCfg := s.Config.Models[model.Definition.GQLDefinition.Name]
		modelCfg.Model = types.TypeString(model.Definition.GoType, nil)
		s.Config.Models[model.Definition.GQLDefinition.Name] = modelCfg
	}

	for _, enum := range b.Enums {
		modelCfg := s.Config.Models[enum.Definition.GQLDefinition.Name]
		modelCfg.Model = types.TypeString(enum.Definition.GoType, nil)
		s.Config.Models[enum.Definition.GQLDefinition.Name] = modelCfg
	}

	return err
}
