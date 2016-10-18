package graphql

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/neelance/graphql-go/example/starwars"
)

type helloWorldResolver struct{}

func (r *helloWorldResolver) Hello() string {
	return "Hello world!"
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
						"name": "Han Solo"
					},
					{
						"name": "Leia Organa"
					},
					{
						"name": "TIE Advanced x1"
					}
				]
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

			got, err := schema.Exec(test.query, "", test.variables)
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
