package graphql

import (
	"bytes"
	"encoding/json"
	"testing"
)

type helloWorldResolver struct{}

func (r *helloWorldResolver) Hello() string {
	return "Hello world!"
}

var starWarsSchema = `
	schema {
		query: Query
	}

	# test comment
	type Query {
		hero: User
		human(id: ID): User
	}

	type User {
		id: String
		name: String
		height: Float
		friends: [User]
	}
`

type starWarsResolver struct{}

func (r *starWarsResolver) Hero() *userResolver {
	return &userResolver{
		id:   "2001",
		name: "R2-D2",
		friends: []*userResolver{
			{name: "Luke Skywalker"},
			{name: "Han Solo"},
			{name: "Leia Organa"},
		},
	}
}

func (r *starWarsResolver) Human(args *struct{ ID string }) *userResolver {
	if args.ID == "1000" {
		return &userResolver{
			name:   "Luke Skywalker",
			height: 1.72,
		}
	}
	return nil
}

type userResolver struct {
	id      string
	name    string
	height  float64
	friends []*userResolver
}

func (r *userResolver) ID() string {
	return r.id
}

func (r *userResolver) Name() string {
	return r.name
}

func (r *userResolver) Height() float64 {
	return r.height
}

func (r *userResolver) Friends() []*userResolver {
	return r.friends
}

var tests = []struct {
	name     string
	schema   string
	resolver interface{}
	query    string
	result   string
}{
	{
		name: "HelloWorld",
		schema: `
			schema {
				query: Query
			}
			
			type Query {
				hello: String
			}
		`,
		resolver: &helloWorldResolver{},
		query: `
			{
				hello
			}
		`,
		result: `
			{
				"hello": "Hello world!"
			}
		`,
	},
	{
		name:     "StarWars1",
		schema:   starWarsSchema,
		resolver: &starWarsResolver{},
		query: `
			{
				hero {
					id
					name
					friends {
						name
					}
				}
			}
		`,
		result: `
			{
				"hero": {
					"id": "2001",
					"name": "R2-D2",
					"friends": [
						{
							"name": "Luke Skywalker"
						},
						{
							"name": "Han Solo"
						},
						{
							"name": "Leia Organa"
						}
					]
				}
			}
		`,
	},
	{
		name:     "StarWars2",
		schema:   starWarsSchema,
		resolver: &starWarsResolver{},
		query: `
			{
				human(id: "1000") {
					name
					height
				}
			}
		`,
		result: `
			{
				"human": {
					"name": "Luke Skywalker",
					"height": 1.72
				}
			}
		`,
	},
}

func TestAll(t *testing.T) {
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			schema, err := NewSchema(test.schema, test.name, test.resolver)
			if err != nil {
				t.Fatal(err)
			}

			got, err := schema.Exec(test.query)
			if err != nil {
				t.Fatal(err)
			}

			want := formatJSON([]byte(test.result))
			if !bytes.Equal(got, want) {
				t.Logf("want: %s", want)
				t.Logf("got:  %s", got)
				t.Fail()
			}
		})
	}
}

func formatJSON(data []byte) []byte {
	var v interface{}
	json.Unmarshal(data, &v)
	b, _ := json.Marshal(v)
	return b
}
