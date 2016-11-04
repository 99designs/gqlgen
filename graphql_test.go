package graphql_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/neelance/graphql-go"
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

type theNumberResolver struct {
	number int32
}

func (r *theNumberResolver) TheNumber() int32 {
	return r.number
}

func (r *theNumberResolver) ChangeTheNumber(args *struct{ NewNumber int32 }) *theNumberResolver {
	r.number = args.NewNumber
	return r
}

type timeResolver struct{}

func (r *timeResolver) AddHour(args *struct{ Time time.Time }) time.Time {
	return args.Time.Add(time.Hour)
}

type Test struct {
	schema    *graphql.Schema
	variables map[string]interface{}
	query     string
	result    string
}

var starwarsSchema = graphql.MustParseSchema(starwars.Schema, &starwars.Resolver{})

func TestHelloWorld(t *testing.T) {
	runTests(t, []*Test{
		{
			schema: graphql.MustParseSchema(`
				schema {
					query: Query
				}
				
				type Query {
					hello: String!
				}
			`, &helloWorldResolver1{}),
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
			schema: graphql.MustParseSchema(`
				schema {
					query: Query
				}
				
				type Query {
					hello: String!
				}
			`, &helloWorldResolver2{}),
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
	})
}

func TestBasic(t *testing.T) {
	runTests(t, []*Test{
		{
			schema: starwarsSchema,
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
	})
}

func TestArguments(t *testing.T) {
	runTests(t, []*Test{
		{
			schema: starwarsSchema,
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
			schema: starwarsSchema,
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
	})
}

func TestAliases(t *testing.T) {
	runTests(t, []*Test{
		{
			schema: starwarsSchema,
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
	})
}

func TestFragments(t *testing.T) {
	runTests(t, []*Test{
		{
			schema: starwarsSchema,
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
	})
}

func TestVariables(t *testing.T) {
	runTests(t, []*Test{
		{
			schema: starwarsSchema,
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
			schema: starwarsSchema,
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
	})
}

func TestSkipDirective(t *testing.T) {
	runTests(t, []*Test{
		{
			schema: starwarsSchema,
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
			schema: starwarsSchema,
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
	})
}

func TestIncludeDirective(t *testing.T) {
	runTests(t, []*Test{
		{
			schema: starwarsSchema,
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
			schema: starwarsSchema,
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
	})
}

func TestInlineFragments(t *testing.T) {
	runTests(t, []*Test{
		{
			schema: starwarsSchema,
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
			schema: starwarsSchema,
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
	})
}

func TestTypeName(t *testing.T) {
	runTests(t, []*Test{
		{
			schema: starwarsSchema,
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
	})
}

func TestConnections(t *testing.T) {
	runTests(t, []*Test{
		{
			schema: starwarsSchema,
			query: `
				{
					hero {
						name
						friendsConnection {
							totalCount
							pageInfo {
								startCursor
								endCursor
								hasNextPage
							}
							edges {
								cursor
								node {
									name
								}
							}
						}
					}
				}
			`,
			result: `
				{
					"hero": {
						"name": "R2-D2",
						"friendsConnection": {
							"totalCount": 3,
							"pageInfo": {
								"startCursor": "Y3Vyc29yMQ==",
								"endCursor": "Y3Vyc29yMw==",
								"hasNextPage": false
							},
							"edges": [
								{
									"cursor": "Y3Vyc29yMQ==",
									"node": {
										"name": "Luke Skywalker"
									}
								},
								{
									"cursor": "Y3Vyc29yMg==",
									"node": {
										"name": "Han Solo"
									}
								},
								{
									"cursor": "Y3Vyc29yMw==",
									"node": {
										"name": "Leia Organa"
									}
								}
							]
						}
					}
				}
			`,
		},

		{
			schema: starwarsSchema,
			query: `
				{
					hero {
						name
						friendsConnection(first: 1, after: "Y3Vyc29yMQ==") {
							totalCount
							pageInfo {
								startCursor
								endCursor
								hasNextPage
							}
							edges {
								cursor
								node {
									name
								}
							}
						}
					}
				}
			`,
			result: `
				{
					"hero": {
						"name": "R2-D2",
						"friendsConnection": {
							"totalCount": 3,
							"pageInfo": {
								"startCursor": "Y3Vyc29yMg==",
								"endCursor": "Y3Vyc29yMg==",
								"hasNextPage": true
							},
							"edges": [
								{
									"cursor": "Y3Vyc29yMg==",
									"node": {
										"name": "Han Solo"
									}
								}
							]
						}
					}
				}
			`,
		},
	})
}

func TestMutation(t *testing.T) {
	runTests(t, []*Test{
		{
			schema: starwarsSchema,
			query: `
				{
					reviews(episode: "JEDI") {
						stars
						commentary
					}
				}
			`,
			result: `
				{
					"reviews": []
				}
			`,
		},

		{
			schema: starwarsSchema,
			query: `
				mutation CreateReviewForEpisode($ep: Episode!, $review: ReviewInput!) {
					createReview(episode: $ep, review: $review) {
						stars
						commentary
					}
				}
			`,
			variables: map[string]interface{}{
				"ep": "JEDI",
				"review": map[string]interface{}{
					"stars":      5,
					"commentary": "This is a great movie!",
				},
			},
			result: `
				{
					"createReview": {
						"stars": 5,
						"commentary": "This is a great movie!"
					}
				}
			`,
		},

		{
			schema: starwarsSchema,
			query: `
				{
					reviews(episode: "JEDI") {
						stars
						commentary
					}
				}
			`,
			result: `
				{
					"reviews": [{
						"stars": 5,
						"commentary": "This is a great movie!"
					}]
				}
			`,
		},
	})
}

func TestIntrospection(t *testing.T) {
	runTests(t, []*Test{
		{
			schema: starwarsSchema,
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
							{ "name": "Boolean" },
							{ "name": "Character" },
							{ "name": "Droid" },
							{ "name": "Episode" },
							{ "name": "Float" },
							{ "name": "FriendsConnection" },
							{ "name": "FriendsEdge" },
							{ "name": "Human" },
							{ "name": "ID" },
							{ "name": "Int" },
							{ "name": "LengthUnit" },
							{ "name": "Mutation" },
							{ "name": "PageInfo" },
							{ "name": "Query" },
							{ "name": "Review" },
							{ "name": "ReviewInput" },
							{ "name": "SearchResult" },
							{ "name": "Starship" },
							{ "name": "String" },
							{ "name": "__Directive" },
							{ "name": "__DirectiveLocation" },
							{ "name": "__EnumValue" },
							{ "name": "__Field" },
							{ "name": "__InputValue" },
							{ "name": "__Schema" },
							{ "name": "__Type" },
							{ "name": "__TypeKind" }
						]
					}
				}
			`,
		},

		{
			schema: starwarsSchema,
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
			schema: starwarsSchema,
			query: `
				{
					a: __type(name: "Droid") {
						name
						kind
						interfaces {
							name
						}
						possibleTypes {
							name
						}
					},
					b: __type(name: "Character") {
						name
						kind
						interfaces {
							name
						}
						possibleTypes {
							name
						}
					}
					c: __type(name: "SearchResult") {
						name
						kind
						interfaces {
							name
						}
						possibleTypes {
							name
						}
					}
				}
			`,
			result: `
				{
					"a": {
						"name": "Droid",
						"kind": "OBJECT",
						"interfaces": [
							{
								"name": "Character"
							}
						],
						"possibleTypes": null
					},
					"b": {
						"name": "Character",
						"kind": "INTERFACE",
						"interfaces": null,
						"possibleTypes": [
							{
								"name": "Human"
							},
							{
								"name": "Droid"
							}
						]
					},
					"c": {
						"name": "SearchResult",
						"kind": "UNION",
						"interfaces": null,
						"possibleTypes": [
							{
								"name": "Human"
							},
							{
								"name": "Droid"
							},
							{
								"name": "Starship"
							}
						]
					}
				}
			`,
		},

		{
			schema: starwarsSchema,
			query: `
				{
					__type(name: "Droid") {
						name
						fields {
							name
							args {
								name
								type {
									name
								}
								defaultValue
							}
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
								"name": "id",
								"args": [],
								"type": {
									"name": null,
									"kind": "NON_NULL"
								}
							},
							{
								"name": "name",
								"args": [],
								"type": {
									"name": null,
									"kind": "NON_NULL"
								}
							},
							{
								"name": "friends",
								"args": [],
								"type": {
									"name": null,
									"kind": "LIST"
								}
							},
							{
								"name": "friendsConnection",
								"args": [
									{
										"name": "first",
										"type": {
											"name": "Int"
										},
										"defaultValue": null
									},
									{
										"name": "after",
										"type": {
											"name": "ID"
										},
										"defaultValue": null
									}
								],
								"type": {
									"name": null,
									"kind": "NON_NULL"
								}
							},
							{
								"name": "appearsIn",
								"args": [],
								"type": {
									"name": null,
									"kind": "NON_NULL"
								}
							},
							{
								"name": "primaryFunction",
								"args": [],
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

		{
			schema: starwarsSchema,
			query: `
				{
					__type(name: "Episode") {
						enumValues {
							name
						}
					}
				}
			`,
			result: `
				{
					"__type": {
						"enumValues": [
							{
								"name": "NEWHOPE"
							},
							{
								"name": "EMPIRE"
							},
							{
								"name": "JEDI"
							}
						]
					}
				}
			`,
		},
	})
}

func TestMutationOrder(t *testing.T) {
	runTests(t, []*Test{
		{
			schema: graphql.MustParseSchema(`
				schema {
					query: Query
					mutation: Mutation
				}

				type Query {
					theNumber: Int!
				}

				type Mutation {
					changeTheNumber(newNumber: Int!): Query
				}
			`, &theNumberResolver{}),
			query: `
				mutation {
					first: changeTheNumber(newNumber: 1) {
						theNumber
					}
					second: changeTheNumber(newNumber: 3) {
						theNumber
					}
					third: changeTheNumber(newNumber: 2) {
						theNumber
					}
				}
			`,
			result: `
				{
					"first": {
						"theNumber": 1
					},
					"second": {
						"theNumber": 3
					},
					"third": {
						"theNumber": 2
					}
				}
			`,
		},
	})
}

func TestTime(t *testing.T) {
	b := graphql.New()
	b.AddCustomScalar("Time", graphql.Time)

	if err := b.Parse(`
		schema {
			query: Query
		}

		type Query {
			addHour(time: Time = "2001-02-03T04:05:06Z"): Time!
		}
	`); err != nil {
		t.Fatal(err)
	}

	schema, err := b.ApplyResolver(&timeResolver{})
	if err != nil {
		t.Fatal(err)
	}

	runTests(t, []*Test{
		{
			schema: schema,
			query: `
				query($t: Time!) {
					a: addHour(time: $t)
					b: addHour
				}
			`,
			variables: map[string]interface{}{
				"t": time.Date(2000, 2, 3, 4, 5, 6, 0, time.UTC),
			},
			result: `
				{
					"a": "2000-02-03T05:05:06Z",
					"b": "2001-02-03T05:05:06Z"
				}
			`,
		},
	})
}

func runTests(t *testing.T, tests []*Test) {
	if len(tests) == 1 {
		runTest(t, tests[0])
		return
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			runTest(t, test)
		})
	}
}

func runTest(t *testing.T, test *Test) {
	result := test.schema.Exec(context.Background(), test.query, "", test.variables)
	if len(result.Errors) != 0 {
		t.Fatal(result.Errors[0])
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
}

func formatJSON(data []byte) []byte {
	var v interface{}
	json.Unmarshal(data, &v)
	b, _ := json.Marshal(v)
	return b
}
