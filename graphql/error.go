package graphql

import (
	"github.com/vektah/gqlgen/neelance/errors"
)

func MarshalErrors(errs []*errors.QueryError) Marshaler {
	res := Array{}
	for _, err := range errs {
		res = append(res, MarshalError(err))
	}
	return res
}

func MarshalError(err *errors.QueryError) Marshaler {
	if err == nil {
		return Null
	}

	errObj := &OrderedMap{}
	errObj.Add("message", MarshalString(err.Message))

	if len(err.Locations) > 0 {
		locations := Array{}
		for _, location := range err.Locations {
			locationObj := &OrderedMap{}
			locationObj.Add("line", MarshalInt(location.Line))
			locationObj.Add("column", MarshalInt(location.Column))

			locations = append(locations, locationObj)
		}

		errObj.Add("locations", locations)
	}
	return errObj
}
