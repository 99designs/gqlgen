package graphql

import (
	"github.com/vektah/gqlgen/neelance/errors"
)

func MarshalErrors(errs []*errors.QueryError) Marshaler {
	res := Array{}

	for _, err := range errs {
		if err == nil {
			res = append(res, Null)
			continue
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
		res = append(res, errObj)
	}

	return res
}
