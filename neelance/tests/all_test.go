package tests

import (
	"os"
	"reflect"
	"sort"
	"testing"

	"encoding/json"

	"github.com/vektah/gqlgen/neelance/errors"
	"github.com/vektah/gqlgen/neelance/query"
	"github.com/vektah/gqlgen/neelance/schema"
	"github.com/vektah/gqlgen/neelance/validation"
)

type Test struct {
	Name   string
	Rule   string
	Schema int
	Query  string
	Errors []*errors.QueryError
}

func TestAll(t *testing.T) {
	f, err := os.Open("testdata/tests.json")
	if err != nil {
		t.Fatal(err)
	}

	var testData struct {
		Schemas []string
		Tests   []*Test
	}
	if err := json.NewDecoder(f).Decode(&testData); err != nil {
		t.Fatal(err)
	}

	schemas := make([]*schema.Schema, len(testData.Schemas))
	for i, schemaStr := range testData.Schemas {
		schemas[i] = schema.New()
		if err := schemas[i].Parse(schemaStr); err != nil {
			t.Fatal(err)
		}
	}

	for _, test := range testData.Tests {
		t.Run(test.Name, func(t *testing.T) {
			d, err := query.Parse(test.Query)
			if err != nil {
				t.Fatal(err)
			}
			errs := validation.Validate(schemas[test.Schema], d)
			got := []*errors.QueryError{}
			for _, err := range errs {
				if err.Rule == test.Rule {
					err.Rule = ""
					got = append(got, err)
				}
			}
			sortLocations(test.Errors)
			sortLocations(got)
			if !reflect.DeepEqual(test.Errors, got) {
				t.Errorf("wrong errors\nexpected: %v\ngot:      %v", test.Errors, got)
			}
		})
	}
}

func sortLocations(errs []*errors.QueryError) {
	for _, err := range errs {
		locs := err.Locations
		sort.Slice(locs, func(i, j int) bool { return locs[i].Before(locs[j]) })
	}
}
