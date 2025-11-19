package modelgen

import (
	"fmt"
	"go/types"
	"log"
	"sort"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/99designs/gqlgen/internal/code"
)

// embeddedInterfaceGenerator generates Base structs for interfaces to enable embedding.
type embeddedInterfaceGenerator struct {
	cfg        *config.Config
	binder     *config.Binder
	schemaType *ast.Definition
	model      *ModelBuild
	graph      *interfaceGraph
}

func newEmbeddedInterfaceGenerator(
	cfg *config.Config,
	binder *config.Binder,
	schemaType *ast.Definition,
	model *ModelBuild,
) *embeddedInterfaceGenerator {
	return &embeddedInterfaceGenerator{
		cfg:        cfg,
		binder:     binder,
		schemaType: schemaType,
		model:      model,
		graph:      newInterfaceGraph(cfg.Schema),
	}
}

// generateAllInterfaceBaseStructs returns Base struct specs ordered with parents before children.
func (g *embeddedInterfaceGenerator) generateAllInterfaceBaseStructs() ([]*baseStructSpec, error) {
	// Filter to only embeddable interfaces (have directive) and not bound to external packages
	var interfaceNames []string
	for name := range g.graph.parentInterfaces {
		// Only include interfaces with directive
		if g.graph.isEmbeddable(name) {
			// Skip interfaces bound to external packages - their Base structs already exist there
			if !g.cfg.Models.UserDefined(name) {
				interfaceNames = append(interfaceNames, name)
			}
		}
	}

	sorted, err := g.graph.topologicalSort(interfaceNames)
	if err != nil {
		return nil, fmt.Errorf("failed to sort interfaces: %w", err)
	}

	var specs []*baseStructSpec
	for _, name := range sorted {
		spec, err := g.generateBaseStructForInterface(g.cfg.Schema.Types[name])
		if err != nil {
			return nil, err
		}
		if spec != nil {
			specs = append(specs, spec)
		}
	}

	return specs, nil
}

// baseStructSpec defines Base struct structure for an interface
type baseStructSpec struct {
	SchemaType           *ast.Definition
	ParentEmbeddings     []types.Type
	FieldsToGenerate     []*ast.FieldDefinition
	ImplementsInterfaces []string
}

func (g *embeddedInterfaceGenerator) generateBaseStructForInterface(
	schemaType *ast.Definition,
) (*baseStructSpec, error) {
	if schemaType.Kind != ast.Interface {
		return nil, fmt.Errorf(
			"generateBaseStructForInterface called on non-interface type: %s",
			schemaType.Name,
		)
	}

	spec := &baseStructSpec{
		SchemaType:           schemaType,
		FieldsToGenerate:     g.graph.getInterfaceOwnFields(schemaType.Name),
		ImplementsInterfaces: []string{schemaType.Name},
	}

	// Get embeddable parents and fields from skipped intermediate parents
	embedInfo := g.graph.getEmbeddingInfo(schemaType.Name)

	if len(embedInfo.Parents) > 1 {
		log.Printf(
			"WARN: Base%s: implements %d interfaces %v (potential diamond problem)",
			schemaType.Name,
			len(embedInfo.Parents),
			embedInfo.Parents,
		)
	}

	for _, parent := range embedInfo.Parents {
		spec.ParentEmbeddings = append(spec.ParentEmbeddings, g.createParentBaseType(parent))
		spec.ImplementsInterfaces = append(spec.ImplementsInterfaces, parent)
	}

	// Add fields from intermediate parents without the directive
	spec.FieldsToGenerate = append(spec.FieldsToGenerate, embedInfo.SkippedFields...)

	return spec, nil
}

func (g *embeddedInterfaceGenerator) createParentBaseType(interfaceName string) types.Type {
	baseName := templates.ToGo(fmt.Sprintf("%s%s", g.cfg.EmbeddedStructsPrefix, interfaceName))

	// Check if interface is bound to external package
	if g.cfg.Models.UserDefined(interfaceName) {
		if models := g.cfg.Models[interfaceName]; len(models.Model) > 0 {
			if extType, err := g.binder.FindTypeFromName(models.Model[0]); err == nil {
				if named, ok := extType.(*types.Named); ok {
					if pkg := named.Obj().Pkg(); pkg != nil {
						if obj := pkg.Scope().Lookup(baseName); obj != nil {
							if typeObj, ok := obj.(*types.TypeName); ok {
								return typeObj.Type()
							}
						}
					}
				}
			}
		}
	}

	// Default: reference local package type
	return types.NewNamed(
		types.NewTypeName(0, g.cfg.Model.Pkg(), baseName, nil),
		types.NewStruct(nil, nil),
		nil,
	)
}

// generateEmbeddedFields returns map: field name -> embedded Base struct (or nil for subsequent
// fields).
// Covariant overrides prevent embedding and require explicit field generation.
func (g *embeddedInterfaceGenerator) generateEmbeddedFields(
	fields []*ast.FieldDefinition,
) map[string]*Field {
	if g.model == nil || g.schemaType.Kind != ast.Object {
		return nil
	}

	covariantInterfaces := g.findInterfacesWithCovariantOverrides(fields)
	result := make(map[string]*Field)
	processed := make(map[string]bool)

	for _, field := range fields {
		interfaceName := g.findInterfaceForField(field)
		if interfaceName == "" || covariantInterfaces[interfaceName] {
			continue
		}

		if processed[interfaceName] {
			result[field.Name] = nil // subsequent field from same interface
		} else {
			result[field.Name] = &Field{Type: g.createEmbeddedBaseType(interfaceName)}
			processed[interfaceName] = true
		}
	}

	return result
}

func (g *embeddedInterfaceGenerator) findInterfacesWithCovariantOverrides(
	fields []*ast.FieldDefinition,
) map[string]bool {
	result := make(map[string]bool)

	for _, implField := range fields {
		for _, interfaceName := range g.schemaType.Interfaces {
			if !g.graph.isEmbeddable(interfaceName) {
				continue
			}

			iface := g.cfg.Schema.Types[interfaceName]
			if iface == nil {
				continue
			}

			for _, ifaceField := range iface.Fields {
				if ifaceField.Name != implField.Name ||
					typesMatch(ifaceField.Type, implField.Type) {
					continue
				}

				if !result[interfaceName] {
					log.Printf(
						"WARN: %s.%s: covariant override %s -> %s (skipping Base%s embedding)",
						g.schemaType.Name,
						implField.Name,
						ifaceField.Type.Name(),
						implField.Type.Name(),
						interfaceName,
					)
				}
				result[interfaceName] = true
				break
			}
		}
	}

	return result
}

// findInterfaceForField returns deepest interface containing this field with matching type.
func (g *embeddedInterfaceGenerator) findInterfaceForField(field *ast.FieldDefinition) string {
	interfaces := g.schemaType.Interfaces
	if len(interfaces) == 0 {
		return ""
	}

	// Sort deepest-first (child interfaces before parent interfaces)
	if len(interfaces) > 1 {
		sorted := make([]string, len(interfaces))
		copy(sorted, interfaces)
		sort.Slice(sorted, func(i, j int) bool {
			depthI := len(g.cfg.Schema.Types[sorted[i]].Interfaces)
			depthJ := len(g.cfg.Schema.Types[sorted[j]].Interfaces)
			return depthI > depthJ
		})
		interfaces = sorted
	}

	for _, ifaceName := range interfaces {
		if iface := g.cfg.Schema.Types[ifaceName]; iface != nil &&
			(iface.Kind == ast.Interface || iface.Kind == ast.Union) {
			if !g.graph.isEmbeddable(ifaceName) {
				continue
			}

			for _, ifaceField := range iface.Fields {
				if ifaceField.Name == field.Name && typesMatch(ifaceField.Type, field.Type) {
					return ifaceName
				}
			}
		}
	}
	return ""
}

// typesMatch checks if two GraphQL types are identical (same base type, nullability, and list
// wrapping).
func typesMatch(a, b *ast.Type) bool {
	if a.Name() != b.Name() || a.NonNull != b.NonNull {
		return false
	}

	// Base type reached
	if a.NamedType != "" && b.NamedType != "" {
		return true
	}

	// Both must be lists or both must not be lists
	if (a.Elem == nil) != (b.Elem == nil) {
		return false
	}

	// Recursively check list element types
	if a.Elem != nil {
		return typesMatch(a.Elem, b.Elem)
	}

	return true
}

func (g *embeddedInterfaceGenerator) createEmbeddedBaseType(interfaceName string) types.Type {
	baseName := templates.ToGo(fmt.Sprintf("%s%s", g.cfg.EmbeddedStructsPrefix, interfaceName))

	// Check if interface is bound to external package
	if g.cfg.Models.UserDefined(interfaceName) {
		if pkgPath, _ := code.PkgAndType(g.cfg.Models[interfaceName].Model[0]); pkgPath != "" {
			if boundType, _ := g.binder.FindTypeFromName(
				pkgPath + "." + baseName,
			); boundType != nil {
				return boundType
			}
		}
	}

	// Default: reference local package type
	return types.NewNamed(
		types.NewTypeName(0, g.cfg.Model.Pkg(), baseName, nil),
		types.NewStruct(nil, nil),
		nil,
	)
}
