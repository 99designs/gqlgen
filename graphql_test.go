package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/neelance/graphql-go/example/starwars"
)

type helloWorldResolver1 struct{}

func (r *helloWorldResolver1) Hello() string {
	return "Hello world!"
}

type helloWorldResolver2 struct{}

func (r *helloWorldResolver2) Hello(ctx context.Context) (string, error) {
	return "Hello world!", nil
}

var tests = []struct {
	name      string
	schema    string
	variables map[string]interface{}
	resolver  interface{}
	query     string
	result    string
}{
	{
		name: "HelloWorld1",
		schema: `
			schema {
				query: Query
			}
			
			type Query {
				hello: String
			}
		`,
		resolver: &helloWorldResolver1{},
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
		name: "HelloWorld2",
		schema: `
			schema {
				query: Query
			}
			
			type Query {
				hello: String
			}
		`,
		resolver: &helloWorldResolver2{},
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
		name:     "StarWarsBasic",
		schema:   starwars.Schema,
		resolver: &starwars.Resolver{},
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
		name:     "StarWarsArguments1",
		schema:   starwars.Schema,
		resolver: &starwars.Resolver{},
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

	{
		name:     "StarWarsArguments2",
		schema:   starwars.Schema,
		resolver: &starwars.Resolver{},
		query: `
			{
				human(id: "1000") {
					name
					height(unit: FOOT)
				}
			}
		`,
		result: `
			{
				"human": {
					"name": "Luke Skywalker",
					"height": 5.6430448
				}
			}
		`,
	},

	{
		name:     "StarWarsAliases",
		schema:   starwars.Schema,
		resolver: &starwars.Resolver{},
		query: `
			{
				empireHero: hero(episode: EMPIRE) {
					name
				}
				jediHero: hero(episode: JEDI) {
					name
				}
			}
		`,
		result: `
			{
				"empireHero": {
					"name": "Luke Skywalker"
				},
				"jediHero": {
					"name": "R2-D2"
				}
			}
		`,
	},

	{
		name:     "StarWarsFragments",
		schema:   starwars.Schema,
		resolver: &starwars.Resolver{},
		query: `
			{
				leftComparison: hero(episode: EMPIRE) {
					...comparisonFields
					...height
				}
				rightComparison: hero(episode: JEDI) {
					...comparisonFields
					...height
				}
			}
			
			fragment comparisonFields on Character {
				name
				appearsIn
				friends {
					name
				}
			}

			fragment height on Human {
				height
			}
		`,
		result: `
			{
				"leftComparison": {
					"name": "Luke Skywalker",
					"appearsIn": [
						"NEWHOPE",
						"EMPIRE",
						"JEDI"
					],
					"friends": [
						{
							"name": "Han Solo"
						},
						{
							"name": "Leia Organa"
						},
						{
							"name": "C-3PO"
						},
						{
							"name": "R2-D2"
						}
					],
					"height": 1.72
				},
				"rightComparison": {
					"name": "R2-D2",
					"appearsIn": [
						"NEWHOPE",
						"EMPIRE",
						"JEDI"
					],
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
		name:     "StarWarsVariables1",
		schema:   starwars.Schema,
		resolver: &starwars.Resolver{},
		query: `
			query HeroNameAndFriends($episode: Episode) {
				hero(episode: $episode) {
					name
				}
			}
		`,
		variables: map[string]interface{}{
			"episode": "JEDI",
		},
		result: `
			{
				"hero": {
					"name": "R2-D2"
				}
			}
		`,
	},

	{
		name:     "StarWarsVariables2",
		schema:   starwars.Schema,
		resolver: &starwars.Resolver{},
		query: `
			query HeroNameAndFriends($episode: Episode) {
				hero(episode: $episode) {
					name
				}
			}
		`,
		variables: map[string]interface{}{
			"episode": "EMPIRE",
		},
		result: `
			{
				"hero": {
					"name": "Luke Skywalker"
				}
			}
		`,
	},

	{
		name:     "StarWarsInclude1",
		schema:   starwars.Schema,
		resolver: &starwars.Resolver{},
		query: `
			query Hero($episode: Episode, $withoutFriends: Boolean!) {
				hero(episode: $episode) {
					name
					friends @skip(if: $withoutFriends) {
						name
					}
				}
			}
		`,
		variables: map[string]interface{}{
			"episode":        "JEDI",
			"withoutFriends": true,
		},
		result: `
			{
				"hero": {
					"name": "R2-D2"
				}
			}
		`,
	},

	{
		name:     "StarWarsInclude2",
		schema:   starwars.Schema,
		resolver: &starwars.Resolver{},
		query: `
			query Hero($episode: Episode, $withoutFriends: Boolean!) {
				hero(episode: $episode) {
					name
					friends @skip(if: $withoutFriends) {
						name
					}
				}
			}
		`,
		variables: map[string]interface{}{
			"episode":        "JEDI",
			"withoutFriends": false,
		},
		result: `
			{
				"hero": {
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
		name:     "StarWarsSkip1",
		schema:   starwars.Schema,
		resolver: &starwars.Resolver{},
		query: `
			query Hero($episode: Episode, $withFriends: Boolean!) {
				hero(episode: $episode) {
					name
					...friendsFragment @include(if: $withFriends)
				}
			}

			fragment friendsFragment on Character {
				friends {
					name
				}
			}
		`,
		variables: map[string]interface{}{
			"episode":     "JEDI",
			"withFriends": false,
		},
		result: `
			{
				"hero": {
					"name": "R2-D2"
				}
			}
		`,
	},

	{
		name:     "StarWarsSkip2",
		schema:   starwars.Schema,
		resolver: &starwars.Resolver{},
		query: `
			query Hero($episode: Episode, $withFriends: Boolean!) {
				hero(episode: $episode) {
					name
					...friendsFragment @include(if: $withFriends)
				}
			}

			fragment friendsFragment on Character {
				friends {
					name
				}
			}
		`,
		variables: map[string]interface{}{
			"episode":     "JEDI",
			"withFriends": true,
		},
		result: `
			{
				"hero": {
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
		name:     "StarWarsInlineFragments1",
		schema:   starwars.Schema,
		resolver: &starwars.Resolver{},
		query: `
			query HeroForEpisode($episode: Episode!) {
				hero(episode: $episode) {
					name
					... on Droid {
						primaryFunction
					}
					... on Human {
						height
					}
				}
			}
		`,
		variables: map[string]interface{}{
			"episode": "JEDI",
		},
		result: `
			{
				"hero": {
					"name": "R2-D2",
					"primaryFunction": "Astromech"
				}
			}
		`,
	},

	{
		name:     "StarWarsInlineFragments2",
		schema:   starwars.Schema,
		resolver: &starwars.Resolver{},
		query: `
			query HeroForEpisode($episode: Episode!) {
				hero(episode: $episode) {
					name
					... on Droid {
						primaryFunction
					}
					... on Human {
						height
					}
				}
			}
		`,
		variables: map[string]interface{}{
			"episode": "EMPIRE",
		},
		result: `
			{
				"hero": {
					"name": "Luke Skywalker",
					"height": 1.72
				}
			}
		`,
	},

	{
		name:     "StarWarsTypeName",
		schema:   starwars.Schema,
		resolver: &starwars.Resolver{},
		query: `
			{
				search(text: "an") {
					__typename
					... on Human {
						name
					}
					... on Droid {
						name
					}
					... on Starship {
						name
					}
				}
			}
		`,
		result: `
			{
				"search": [
					{
						"__typename": "Human",
						"name": "Han Solo"
					},
					{
						"__typename": "Human",
						"name": "Leia Organa"
					},
					{
						"__typename": "Starship",
						"name": "TIE Advanced x1"
					}
				]
			}
		`,
	},

	{
		name:     "StarWarsIntrospection1",
		schema:   starwars.Schema,
		resolver: &starwars.Resolver{},
		query: `
			{
				__schema {
					types {
						name
					}
				}
			}
		`,
		result: `
			{
				"__schema": {
					"types": [
						{ "name": "Character" },
						{ "name": "Droid" },
						{ "name": "Episode" },
						{ "name": "FriendsConnection" },
						{ "name": "FriendsEdge" },
						{ "name": "Human" },
						{ "name": "LengthUnit" },
						{ "name": "Mutation" },
						{ "name": "PageInfo" },
						{ "name": "Query" },
						{ "name": "Review" },
						{ "name": "ReviewInput" },
						{ "name": "SearchResult" },
						{ "name": "Starship" },
						{ "name": "__Directive" },
						{ "name": "__DirectiveLocation" },
						{ "name": "__EnumValue" },
						{ "name": "__Field" },
						{ "name": "__InputValue" },
						{ "name": "__Schema" },
						{ "name": "__Type" },
						{ "name": "__TypeKind" },
						{ "name": "Int" },
						{ "name": "Float" },
						{ "name": "String" },
						{ "name": "Boolean" },
						{ "name": "ID" }
					]
				}
			}
		`,
	},

	{
		name:     "StarWarsIntrospection2",
		schema:   starwars.Schema,
		resolver: &starwars.Resolver{},
		query: `
			{
				__schema {
					queryType {
						name
					}
				}
			}
		`,
		result: `
			{
				"__schema": {
					"queryType": {
						"name": "Query"
					}
				}
			}
		`,
	},

	{
		name:     "StarWarsIntrospection3",
		schema:   starwars.Schema,
		resolver: &starwars.Resolver{},
		query: `
			{
				a: __type(name: "Droid") {
					name
					kind
				},
				b: __type(name: "Character") {
					name
					kind
				}
			}
		`,
		result: `
			{
				"a": {
					"name": "Droid",
					"kind": "OBJECT"
				},
				"b": {
					"name": "Character",
					"kind": "INTERFACE"
				}
			}
		`,
	},

	{
		name:     "StarWarsIntrospection4",
		schema:   starwars.Schema,
		resolver: &starwars.Resolver{},
		query: `
			{
				__type(name: "Droid") {
					name
					fields {
						name
						type {
							name
							kind
						}
					}
				}
			}
		`,
		result: `
			{
				"__type": {
					"name": "Droid",
					"fields": [
						{
							"name": "appearsIn",
							"type": {
								"name": null,
								"kind": "NON_NULL"
							}
						},
						{
							"name": "friends",
							"type": {
								"name": null,
								"kind": "LIST"
							}
						},
						{
							"name": "friendsConnection",
							"type": {
								"name": null,
								"kind": "NON_NULL"
							}
						},
						{
							"name": "id",
							"type": {
								"name": null,
								"kind": "NON_NULL"
							}
						},
						{
							"name": "name",
							"type": {
								"name": null,
								"kind": "NON_NULL"
							}
						},
						{
							"name": "primaryFunction",
							"type": {
								"name": "String",
								"kind": "SCALAR"
							}
						}
					]
				}
			}
		`,
	},
}

func TestAll(t *testing.T) {
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			schema, err := ParseSchema(test.schema, test.resolver)
			if err != nil {
				t.Fatal(err)
			}

			result := schema.Exec(context.Background(), test.query, "", test.variables)
			if err != nil {
				t.Fatal(err)
			}

			got, err := json.Marshal(result.Data)
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
