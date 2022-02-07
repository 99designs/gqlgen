package graph

import "github.com/99designs/gqlgen/_examples/federation/products/graph/model"

var hats = []*model.Product{
	{
		ID: "111",
		Manufacturer: &model.Manufacturer{
			ID:   "1234",
			Name: "Millinery 1234",
		},
		Upc:   "top-1",
		Name:  "Trilby",
		Price: 11,
	},
	{
		ID: "222",
		Manufacturer: &model.Manufacturer{
			ID:   "2345",
			Name: "Millinery 2345",
		},
		Upc:   "top-2",
		Name:  "Fedora",
		Price: 22,
	},
	{
		ID: "333",
		Manufacturer: &model.Manufacturer{
			ID:   "2345",
			Name: "Millinery 2345",
		},
		Upc:   "top-3",
		Name:  "Boater",
		Price: 33,
	},
}
