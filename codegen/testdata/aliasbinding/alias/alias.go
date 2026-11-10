package alias

import "github.com/99designs/gqlgen/codegen/testdata/aliasbinding/model"

type Access = model.Access

type Aliased struct{ Access []Access }
