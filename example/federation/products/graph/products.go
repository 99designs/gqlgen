package graph

import "github.com/99designs/gqlgen/example/federation/products/graph/model"

var hats = []*model.Product{
	{
		Upc:   "top-1",
		Name:  "Trilby",
		Price: 11,
	},
	{
		Upc:   "top-2",
		Name:  "Fedora",
		Price: 22,
	},
	{
		Upc:   "top-3",
		Name:  "Boater",
		Price: 33,
	},
}
