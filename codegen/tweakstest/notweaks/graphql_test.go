package notweaks_test

import (
	"reflect"
	"testing"

	"github.com/99designs/gqlgen/codegen/tweakstest/notweaks"
	"github.com/99designs/gqlgen/handler"
)

func TestNoTweaks(t *testing.T) {
	handler := handler.GraphQL(notweaks.NewExecutableSchema(notweaks.NewConfig()))
	if handler == nil {
		t.Fatal()
	}

	t.Log("That it compiles with the resolvers is test passed enough")

	if reflect.TypeOf((notweaks.Page{}).Tags).Elem().Kind() != reflect.Ptr {
		t.Fatal("Generated Page.Tags should be a pointer")
	}
}
