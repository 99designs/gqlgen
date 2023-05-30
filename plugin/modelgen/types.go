package modelgen

import (
	"go/types"
	"strings"
)

// buildType constructs a types.Type for the given string (using the syntax
// from the extra field config Type field).
func buildType(typeString string) types.Type {
	switch {
	case typeString[0] == '*':
		return types.NewPointer(buildType(typeString[1:]))
	case strings.HasPrefix(typeString, "[]"):
		return types.NewSlice(buildType(typeString[2:]))
	default:
		return buildNamedType(typeString)
	}
}

// buildNamedType returns the specified named or builtin type.
//
// Note that we don't look up the full types.Type object from the appropriate
// package -- gqlgen doesn't give us the package-map we'd need to do so.
// Instead we construct a placeholder type that has all the fields gqlgen
// wants. This is roughly what gqlgen itself does, anyway:
// https://github.com/99designs/gqlgen/blob/master/plugin/modelgen/models.go#L119
func buildNamedType(fullName string) types.Type {
	dotIndex := strings.LastIndex(fullName, ".")
	if dotIndex == -1 { // builtinType
		return types.Universe.Lookup(fullName).Type()
	}

	// type is pkg.Name
	pkgPath := fullName[:dotIndex]
	typeName := fullName[dotIndex+1:]

	pkgName := pkgPath
	slashIndex := strings.LastIndex(pkgPath, "/")
	if slashIndex != -1 {
		pkgName = pkgPath[slashIndex+1:]
	}

	pkg := types.NewPackage(pkgPath, pkgName)
	// gqlgen doesn't use some of the fields, so we leave them 0/nil
	return types.NewNamed(types.NewTypeName(0, pkg, typeName, nil), nil, nil)
}
