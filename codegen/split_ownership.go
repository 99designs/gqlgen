package codegen

import (
	"fmt"
	"sort"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/vektah/gqlparser/v2/ast"
)

type splitOwnershipPlanner struct {
	FieldOwner        map[string]string
	ArgsOwner         map[string]string
	FieldContextOwner map[string]string
	ComplexityOwner   map[string]string
	InputOwner        map[string]string
	CodecOwner        map[string]string

	FieldOwnerKeys        []string
	ArgsOwnerKeys         []string
	FieldContextOwnerKeys []string
	ComplexityOwnerKeys   []string
	InputOwnerKeys        []string
	CodecOwnerKeys        []string
}

func planSplitOwnership(data *Data) (*splitOwnershipPlanner, error) {
	builds := map[string]*Data{}
	if err := addObjects(data, &builds); err != nil {
		return nil, err
	}

	filenames := make([]string, 0, len(builds))
	for filename, build := range builds {
		if filename == "" || build == nil || len(build.Objects) == 0 {
			continue
		}
		filenames = append(filenames, filename)
	}
	sort.Strings(filenames)

	filenameToShard := make(map[string]string, len(filenames))
	usedShardNames := map[string]string{}
	for _, filename := range filenames {
		filenameToShard[filename] = splitShardName(filename, builds[filename], usedShardNames)
	}

	planner := &splitOwnershipPlanner{
		FieldOwner:        map[string]string{},
		ArgsOwner:         map[string]string{},
		FieldContextOwner: map[string]string{},
		ComplexityOwner:   map[string]string{},
		InputOwner:        map[string]string{},
		CodecOwner:        map[string]string{},
	}

	for _, filename := range filenames {
		build := builds[filename]
		shard := filenameToShard[filename]
		if err := planner.addBuild(build, shard); err != nil {
			return nil, err
		}
	}

	planner.FieldOwnerKeys = sortedOwnershipKeys(planner.FieldOwner)
	planner.ArgsOwnerKeys = sortedOwnershipKeys(planner.ArgsOwner)
	planner.FieldContextOwnerKeys = sortedOwnershipKeys(planner.FieldContextOwner)
	planner.ComplexityOwnerKeys = sortedOwnershipKeys(planner.ComplexityOwner)
	planner.planInputOwnership(data, builds, filenameToShard)
	planner.InputOwnerKeys = sortedOwnershipKeys(planner.InputOwner)
	planner.planCodecOwnership(data, builds, filenameToShard)
	planner.CodecOwnerKeys = sortedOwnershipKeys(planner.CodecOwner)

	return planner, nil
}

func (p *splitOwnershipPlanner) planCodecOwnership(data *Data, builds map[string]*Data, filenameToShard map[string]string) {
	codecConsumers := map[string]map[string]struct{}{}

	for filename, build := range builds {
		shard := filenameToShard[filename]
		if shard == "" || build == nil {
			continue
		}

		for _, object := range build.Objects {
			for _, field := range object.Fields {
				addCodecConsumer(codecConsumers, field.TypeReference, shard)
				for _, arg := range field.Args {
					addCodecConsumer(codecConsumers, arg.TypeReference, shard)
				}
			}
		}
	}

	for _, input := range data.Inputs {
		owner := p.InputOwner[input.Name]
		if owner == "" {
			owner = "common"
		}

		for _, field := range input.Fields {
			addCodecConsumer(codecConsumers, field.TypeReference, owner)
		}
	}

	referencedTypeKeys := make([]string, 0, len(data.ReferencedTypes))
	for key := range data.ReferencedTypes {
		referencedTypeKeys = append(referencedTypeKeys, key)
	}
	sort.Strings(referencedTypeKeys)

	for _, key := range referencedTypeKeys {
		ref := data.ReferencedTypes[key]
		if ref == nil {
			continue
		}

		if marshal := ref.MarshalFunc(); marshal != "" {
			p.CodecOwner[marshal] = smallestCodecConsumer(codecConsumers[marshal])
		}
		if unmarshal := ref.UnmarshalFunc(); unmarshal != "" {
			p.CodecOwner[unmarshal] = smallestCodecConsumer(codecConsumers[unmarshal])
		}
	}

	for codec, consumers := range codecConsumers {
		if _, exists := p.CodecOwner[codec]; exists {
			continue
		}
		p.CodecOwner[codec] = smallestCodecConsumer(consumers)
	}
}

func addCodecConsumer(consumers map[string]map[string]struct{}, ref *config.TypeReference, shard string) {
	if ref == nil || ref.Definition == nil || ref.GQL == nil || ref.GO == nil || shard == "" {
		return
	}

	if marshal := ref.MarshalFunc(); marshal != "" {
		if consumers[marshal] == nil {
			consumers[marshal] = map[string]struct{}{}
		}
		consumers[marshal][shard] = struct{}{}
	}

	if unmarshal := ref.UnmarshalFunc(); unmarshal != "" {
		if consumers[unmarshal] == nil {
			consumers[unmarshal] = map[string]struct{}{}
		}
		consumers[unmarshal][shard] = struct{}{}
	}
}

func smallestCodecConsumer(consumers map[string]struct{}) string {
	if len(consumers) == 0 {
		return "common"
	}

	shards := make([]string, 0, len(consumers))
	for shard := range consumers {
		shards = append(shards, shard)
	}
	sort.Strings(shards)
	return shards[0]
}

func (p *splitOwnershipPlanner) planInputOwnership(data *Data, builds map[string]*Data, filenameToShard map[string]string) {
	inputDeps := buildInputDependencies(data)
	inputConsumers := map[string]map[string]struct{}{}

	for filename, build := range builds {
		shard := filenameToShard[filename]
		if shard == "" || build == nil {
			continue
		}

		for _, object := range build.Objects {
			for _, field := range object.Fields {
				for _, arg := range field.Args {
					if arg.TypeReference == nil {
						continue
					}

					inputName, ok := inputDefinitionName(arg.TypeReference.Definition)
					if !ok {
						continue
					}

					for _, consumed := range expandInputDependencies(inputName, inputDeps) {
						if inputConsumers[consumed] == nil {
							inputConsumers[consumed] = map[string]struct{}{}
						}
						inputConsumers[consumed][shard] = struct{}{}
					}
				}
			}
		}
	}

	inputs := append(Objects(nil), data.Inputs...)
	sort.Slice(inputs, func(i, j int) bool {
		return inputs[i].Name < inputs[j].Name
	})

	for _, input := range inputs {
		consumers := inputConsumers[input.Name]
		if len(consumers) == 0 {
			p.InputOwner[input.Name] = "common"
			continue
		}

		shards := make([]string, 0, len(consumers))
		for shard := range consumers {
			shards = append(shards, shard)
		}
		sort.Strings(shards)
		p.InputOwner[input.Name] = shards[0]
	}
}

func buildInputDependencies(data *Data) map[string][]string {
	deps := map[string][]string{}

	for _, input := range data.Inputs {
		if input == nil {
			continue
		}

		depSet := map[string]struct{}{}
		for _, field := range input.Fields {
			if field.TypeReference == nil {
				continue
			}

			name, ok := inputDefinitionName(field.TypeReference.Definition)
			if !ok || name == input.Name {
				continue
			}
			depSet[name] = struct{}{}
		}

		depList := make([]string, 0, len(depSet))
		for name := range depSet {
			depList = append(depList, name)
		}
		sort.Strings(depList)
		deps[input.Name] = depList
	}

	return deps
}

func expandInputDependencies(root string, deps map[string][]string) []string {
	visited := map[string]struct{}{}
	stack := []string{root}

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if _, seen := visited[current]; seen {
			continue
		}

		visited[current] = struct{}{}
		for _, next := range deps[current] {
			if _, seen := visited[next]; !seen {
				stack = append(stack, next)
			}
		}
	}

	expanded := make([]string, 0, len(visited))
	for name := range visited {
		expanded = append(expanded, name)
	}
	sort.Strings(expanded)
	return expanded
}

func inputDefinitionName(definition *ast.Definition) (string, bool) {
	if definition == nil || definition.Kind != ast.InputObject {
		return "", false
	}

	return definition.Name, true
}

func (p *splitOwnershipPlanner) addBuild(build *Data, shard string) error {
	objects := append(Objects(nil), build.Objects...)
	sort.Slice(objects, func(i, j int) bool {
		return objects[i].Name < objects[j].Name
	})

	for _, object := range objects {
		fields := append([]*Field(nil), object.Fields...)
		sort.Slice(fields, func(i, j int) bool {
			return fields[i].Name < fields[j].Name
		})

		for _, field := range fields {
			key := object.Name + "." + field.Name

			if err := setOwnedKey(p.FieldOwner, key, shard, "field"); err != nil {
				return err
			}
			if err := setOwnedKey(p.ComplexityOwner, key, shard, "complexity"); err != nil {
				return err
			}

			if argsFunc := field.ArgsFunc(); argsFunc != "" {
				if err := setOwnedKey(p.ArgsOwner, argsFunc, shard, "args"); err != nil {
					return err
				}
			}

			if err := setOwnedKey(p.FieldContextOwner, field.FieldContextFunc(), shard, "field context"); err != nil {
				return err
			}
		}
	}

	return nil
}

func setOwnedKey(owners map[string]string, key, shard, ownerType string) error {
	if current, exists := owners[key]; exists {
		if current != shard {
			return fmt.Errorf("conflicting %s ownership for %q: %q vs %q", ownerType, key, current, shard)
		}
		return nil
	}

	owners[key] = shard
	return nil
}

func sortedOwnershipKeys(owners map[string]string) []string {
	keys := make([]string, 0, len(owners))
	for key := range owners {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
