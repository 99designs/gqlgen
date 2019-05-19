package structvalslices_test

import (
	"reflect"
	"testing"

	"github.com/99designs/gqlgen/codegen/tweakstest/structvalslices"
	"github.com/99designs/gqlgen/handler"
)

func TestNoTweaks(t *testing.T) {
	handler := handler.GraphQL(structvalslices.NewExecutableSchema(structvalslices.NewConfig()))
	if handler == nil {
		t.Fatal()
	}

	t.Log("That it compiles with the resolvers is test passed enough")

	if reflect.TypeOf((structvalslices.Page{}).Tags).Elem().Kind() != reflect.Struct {
		t.Fatal("Generated Page.Tags should be a pointer")
	}
}
