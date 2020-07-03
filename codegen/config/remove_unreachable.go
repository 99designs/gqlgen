package config

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

type reachabilityCleaner struct {
	schema    *ast.Schema
	reachable map[string]*ast.Definition
	queue     []*ast.Definition
}

func (r *reachabilityCleaner) addToQueue(def *ast.Definition) {
	// simplify error handling by acceping nil and ignoring it
	if def == nil {
		return
	}
	if _, ok := r.reachable[def.Name]; !ok {
		r.queue = append(r.queue, def)
	}
	r.reachable[def.Name] = r.schema.Types[def.Name]
}

func (r *reachabilityCleaner) findReachableTypes() map[string]*ast.Definition {
	// Mark all types reachable from the current queue of definitions as reachable
	for len(r.queue) > 0 {
		// pop from stack
		currentType := r.queue[0]
		r.queue = r.queue[1:]

		referencedFromCurrent := []*ast.Definition{}
		for _, f := range currentType.Fields {
			referencedFromCurrent = append(referencedFromCurrent, r.schema.Types[f.Type.Name()])
			for _, arg := range f.Arguments {
				referencedFromCurrent = append(referencedFromCurrent, r.schema.Types[arg.Type.Name()])
			}
		}
		for _, def := range referencedFromCurrent {
			// If this type hasn't been seen before, make sure we expand it by adding to the queue
			r.addToQueue(def)
			// When adding a union or enum to the queue, make sure we also add all its possible implementations
			for _, pt := range r.schema.PossibleTypes[def.Name] {
				r.addToQueue(pt)
			}
			// When adding an implementation of an interface to the queue, make sure we also add the interface it implements
			for _, pt := range r.schema.Implements[def.Name] {
				r.addToQueue(pt)
			}
		}
	}

	// Mark all double underscore types as reachable because they are used for introspection
	for _, d := range r.schema.Types {
		if strings.HasPrefix(d.Name, "__") {
			r.reachable[d.Name] = d
			for _, pt := range r.schema.PossibleTypes[d.Name] {
				r.reachable[pt.Name] = pt
			}
		}
	}
	return r.reachable
}

func calculateAndWarnOnUnreachableTypes(reachableTypes, allTypes map[string]*ast.Definition) {
	unreachableTypes := map[string]struct{}{}
	for all := range allTypes {
		unreachableTypes[all] = struct{}{}
	}
	for r := range reachableTypes {
		delete(unreachableTypes, r)
	}

	if len(unreachableTypes) > 0 {
		unreachableTypesList := []string{}
		for t := range unreachableTypes {
			unreachableTypesList = append(unreachableTypesList, t)
		}
		sort.Strings(unreachableTypesList)
		fmt.Printf("Warning: unreachable types: %s\n", strings.Join(unreachableTypesList, ", "))
	}
}

func removeUnreachableTypes(a *ast.Schema) {
	rc := reachabilityCleaner{
		reachable: map[string]*ast.Definition{},
		schema:    a,
	}
	rc.addToQueue(a.Query)
	rc.addToQueue(a.Mutation)
	rc.addToQueue(a.Subscription)
	// Entity is a fake type used in the federation plugin to generate resolver interfaces to be implemented
	// TODO: find a better way to make sure it's considered reachable
	rc.addToQueue(a.Types["Entity"])
	// All directive arguments are considered reachable for now. No need to clean these up as aggressively.
	for _, dd := range a.Directives {
		for _, arg := range dd.Arguments {
			rc.addToQueue(a.Types[arg.Type.Name()])
		}
	}

	reachable := rc.findReachableTypes()

	calculateAndWarnOnUnreachableTypes(reachable, a.Types)

	a.Types = reachable
}
