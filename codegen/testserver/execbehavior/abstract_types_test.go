package execbehavior

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInterfaceAndUnionValues(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.Animals = func(ctx context.Context) ([]Animal, error) {
		return []Animal{
			Dog{Name: "rex", Barks: true},
			&Dog{Name: "fido"},
			&Cat{Name: "tom", Lives: 9},
			(*Cat)(nil),
			(*Dog)(nil),
			nil,
		}, nil
	}
	resolvers.QueryResolver.Pets = func(ctx context.Context) ([]Pet, error) {
		return []Pet{Dog{Name: "rex"}, &Cat{Name: "tom", Lives: 9}}, nil
	}
	resolvers.QueryResolver.StrangeAnimal = func(ctx context.Context, kind string) (Animal, error) {
		if kind == "robot" {
			return Robot{Model: "k9"}, nil
		}
		return Ghost{}, nil
	}
	srv := newServer(resolvers, DirectiveRoot{})

	t.Run("interface", func(t *testing.T) {
		got := post(t, srv, `{ animals {
			__typename name
			... on Dog { barks }
			... on Cat { lives }
		} }`)
		// The typed nil *Cat and *Dog and the nil interface are null.
		require.JSONEq(t, `{"data":{"animals":[
			{"__typename":"Dog","name":"rex","barks":true},
			{"__typename":"Dog","name":"fido","barks":false},
			{"__typename":"Cat","name":"tom","lives":9},
			null,null,null
		]}}`, got)
	})

	t.Run("union", func(t *testing.T) {
		got := post(t, srv, `{ pets {
			__typename
			... on Dog { name }
			... on Cat { name lives }
		} }`)
		require.JSONEq(t, `{"data":{"pets":[
			{"__typename":"Dog","name":"rex"},
			{"__typename":"Cat","name":"tom","lives":9}
		]}}`, got)
	})

	t.Run("Go type outside the schema that marshals itself", func(t *testing.T) {
		got := post(t, srv, `{ strangeAnimal(kind: "robot") { name } }`)
		require.JSONEq(t, `{"data":{"strangeAnimal":"robot k9"}}`, got)
	})

	t.Run("Go type outside the schema that cannot be marshaled", func(t *testing.T) {
		got := post(t, srv, `{ strangeAnimal(kind: "ghost") { name } }`)
		require.JSONEq(t, `{"data":{"strangeAnimal":null},"errors":[
			{"message":"internal system error","path":["strangeAnimal"]}]}`, got)
	})
}
