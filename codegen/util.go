package codegen

import (
	"fmt"
	"go/types"
	"strings"
)

func findGoNamedType(def types.Type) (*types.Named, error) {
	if def == nil {
		return nil, nil
	}

	namedType, ok := def.(*types.Named)
	//nolint:staticcheck // yes, it is bad to end in newline here
	if !ok {
		return nil, fmt.Errorf(
			"expected %s to be a named type, instead found %T\n",
			def.String(),
			def,
		)
	}

	return namedType, nil
}

func findGoInterface(def types.Type) (*types.Interface, error) {
	if def == nil {
		return nil, nil
	}
	namedType, err := findGoNamedType(def)
	if err != nil {
		return nil, err
	}
	if namedType == nil {
		return nil, nil
	}

	underlying, ok := namedType.Underlying().(*types.Interface)
	if !ok {
		return nil, fmt.Errorf(
			"expected %s to be a named interface, instead found %s",
			def.String(),
			namedType.String(),
		)
	}

	return underlying, nil
}

// equalFieldName reports whether source and target name the same field, ignoring
// underscores and case.
//
// source may be a struct tag value, which encoding/json lets carry options after
// the name ("name,omitempty"). The name is everything before the first comma, so
// reading it that way ignores every option rather than only the ones gqlgen
// happens to emit: modelgen writes "omitzero" and "omitempty,omitzero" too, and
// matching those literally made a tagged field unbindable. Go field names, the
// other kind of source, cannot contain a comma, so the split cannot affect them.
func equalFieldName(source, target string) bool {
	source, _, _ = strings.Cut(source, ",")
	source = strings.ReplaceAll(source, "_", "")
	target = strings.ReplaceAll(target, "_", "")
	return strings.EqualFold(source, target)
}
