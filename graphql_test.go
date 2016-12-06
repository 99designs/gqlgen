package graphql_test

import (
	"context"
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

var starwarsSchema = graphql.MustParseSchema(starwars.Schema, &starwars.Resolver{})

func TestHelloWorld(t *testing.T) {
	graphql.RunTests(t, []*graphql.Test{
		{
			Schema: graphql.MustParseSchema(`
				schema {
					query: Query
				}
				
				type Query {
					hello: String!
				}
			`, &helloWorldResolver1{}),
			Query: `
				{
					hello
				}
			`,
			ExpectedResult: `
				{
					"hello": "Hello world!"
				}
			`,
		},

		{
			Schema: graphql.MustParseSchema(`
				schema {
					query: Query
				}
				
				type Query {
					hello: String!
				}
			`, &helloWorldResolver2{}),
			Query: `
				{
					hello
				}
			`,
			ExpectedResult: `
				{
					"hello": "Hello world!"
				}
			`,
		},
	})
}

func TestBasic(t *testing.T) {
	graphql.RunTests(t, []*graphql.Test{
		{
			Schema: starwarsSchema,
			Query: `
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
			ExpectedResult: `
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
	graphql.RunTests(t, []*graphql.Test{
		{
			Schema: starwarsSchema,
			Query: `
				{
					human(id: "1000") {
						name
						height
					}
				}
			`,
			ExpectedResult: `
				{
					"human": {
						"name": "Luke Skywalker",
						"height": 1.72
					}
				}
			`,
		},

		{
			Schema: starwarsSchema,
			Query: `
				{
					human(id: "1000") {
						name
						height(unit: FOOT)
					}
				}
			`,
			ExpectedResult: `
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
	graphql.RunTests(t, []*graphql.Test{
		{
			Schema: starwarsSchema,
			Query: `
				{
					empireHero: hero(episode: EMPIRE) {
						name
					}
					jediHero: hero(episode: JEDI) {
						name
					}
				}
			`,
			ExpectedResult: `
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
	graphql.RunTests(t, []*graphql.Test{
		{
			Schema: starwarsSchema,
			Query: `
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
			ExpectedResult: `
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
	graphql.RunTests(t, []*graphql.Test{
		{
			Schema: starwarsSchema,
			Query: `
				query HeroNameAndFriends($episode: Episode) {
					hero(episode: $episode) {
						name
					}
				}
			`,
			Variables: map[string]interface{}{
				"episode": "JEDI",
			},
			ExpectedResult: `
				{
					"hero": {
						"name": "R2-D2"
					}
				}
			`,
		},

		{
			Schema: starwarsSchema,
			Query: `
				query HeroNameAndFriends($episode: Episode) {
					hero(episode: $episode) {
						name
					}
				}
			`,
			Variables: map[string]interface{}{
				"episode": "EMPIRE",
			},
			ExpectedResult: `
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
	graphql.RunTests(t, []*graphql.Test{
		{
			Schema: starwarsSchema,
			Query: `
				query Hero($episode: Episode, $withoutFriends: Boolean!) {
					hero(episode: $episode) {
						name
						friends @skip(if: $withoutFriends) {
							name
						}
					}
				}
			`,
			Variables: map[string]interface{}{
				"episode":        "JEDI",
				"withoutFriends": true,
			},
			ExpectedResult: `
				{
					"hero": {
						"name": "R2-D2"
					}
				}
			`,
		},

		{
			Schema: starwarsSchema,
			Query: `
				query Hero($episode: Episode, $withoutFriends: Boolean!) {
					hero(episode: $episode) {
						name
						friends @skip(if: $withoutFriends) {
							name
						}
					}
				}
			`,
			Variables: map[string]interface{}{
				"episode":        "JEDI",
				"withoutFriends": false,
			},
			ExpectedResult: `
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
	graphql.RunTests(t, []*graphql.Test{
		{
			Schema: starwarsSchema,
			Query: `
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
			Variables: map[string]interface{}{
				"episode":     "JEDI",
				"withFriends": false,
			},
			ExpectedResult: `
				{
					"hero": {
						"name": "R2-D2"
					}
				}
			`,
		},

		{
			Schema: starwarsSchema,
			Query: `
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
			Variables: map[string]interface{}{
				"episode":     "JEDI",
				"withFriends": true,
			},
			ExpectedResult: `
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
	graphql.RunTests(t, []*graphql.Test{
		{
			Schema: starwarsSchema,
			Query: `
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
			Variables: map[string]interface{}{
				"episode": "JEDI",
			},
			ExpectedResult: `
				{
					"hero": {
						"name": "R2-D2",
						"primaryFunction": "Astromech"
					}
				}
			`,
		},

		{
			Schema: starwarsSchema,
			Query: `
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
			Variables: map[string]interface{}{
				"episode": "EMPIRE",
			},
			ExpectedResult: `
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
	graphql.RunTests(t, []*graphql.Test{
		{
			Schema: starwarsSchema,
			Query: `
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
			ExpectedResult: `
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
			Schema: starwarsSchema,
			Query: `
				{
					human(id: "1000") {
						__typename
						name
					}
				}
			`,
			ExpectedResult: `
				{
					"human": {
						"__typename": "Human",
						"name": "Luke Skywalker"
					}
				}
			`,
		},
	})
}

func TestConnections(t *testing.T) {
	graphql.RunTests(t, []*graphql.Test{
		{
			Schema: starwarsSchema,
			Query: `
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
			ExpectedResult: `
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
			Schema: starwarsSchema,
			Query: `
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
					},
					moreFriends: hero {
						name
						friendsConnection(first: 1, after: "Y3Vyc29yMg==") {
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
			ExpectedResult: `
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
					},
					"moreFriends": {
						"name": "R2-D2",
						"friendsConnection": {
							"totalCount": 3,
							"pageInfo": {
								"startCursor": "Y3Vyc29yMw==",
								"endCursor": "Y3Vyc29yMw==",
								"hasNextPage": false
							},
							"edges": [
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
	})
}

func TestMutation(t *testing.T) {
	graphql.RunTests(t, []*graphql.Test{
		{
			Schema: starwarsSchema,
			Query: `
				{
					reviews(episode: "JEDI") {
						stars
						commentary
					}
				}
			`,
			ExpectedResult: `
				{
					"reviews": []
				}
			`,
		},

		{
			Schema: starwarsSchema,
			Query: `
				mutation CreateReviewForEpisode($ep: Episode!, $review: ReviewInput!) {
					createReview(episode: $ep, review: $review) {
						stars
						commentary
					}
				}
			`,
			Variables: map[string]interface{}{
				"ep": "JEDI",
				"review": map[string]interface{}{
					"stars":      5,
					"commentary": "This is a great movie!",
				},
			},
			ExpectedResult: `
				{
					"createReview": {
						"stars": 5,
						"commentary": "This is a great movie!"
					}
				}
			`,
		},

		{
			Schema: starwarsSchema,
			Query: `
				mutation CreateReviewForEpisode($ep: Episode!, $review: ReviewInput!) {
					createReview(episode: $ep, review: $review) {
						stars
						commentary
					}
				}
			`,
			Variables: map[string]interface{}{
				"ep": "EMPIRE",
				"review": map[string]interface{}{
					"stars": float64(4),
				},
			},
			ExpectedResult: `
				{
					"createReview": {
						"stars": 4,
						"commentary": null
					}
				}
			`,
		},

		{
			Schema: starwarsSchema,
			Query: `
				{
					reviews(episode: "JEDI") {
						stars
						commentary
					}
				}
			`,
			ExpectedResult: `
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
	graphql.RunTests(t, []*graphql.Test{
		{
			Schema: starwarsSchema,
			Query: `
				{
					__schema {
						types {
							name
						}
					}
				}
			`,
			ExpectedResult: `
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
			Schema: starwarsSchema,
			Query: `
				{
					__schema {
						queryType {
							name
						}
					}
				}
			`,
			ExpectedResult: `
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
			Schema: starwarsSchema,
			Query: `
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
			ExpectedResult: `
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
			Schema: starwarsSchema,
			Query: `
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
			ExpectedResult: `
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
			Schema: starwarsSchema,
			Query: `
				{
					__type(name: "Episode") {
						enumValues {
							name
						}
					}
				}
			`,
			ExpectedResult: `
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

		{
			Schema: starwarsSchema,
			Query: `
				{
					__schema {
						directives {
							name
							description
							locations
							args {
								name
								description
								type {
									kind
									ofType {
										kind
										name
									}
								}
							}
						}
					}
				}
			`,
			ExpectedResult: `
				{
						"__schema": {
							"directives": [
								{
									"name": "skip",
									"description": "Directs the executor to skip this field or fragment when the ` + "`" + `if` + "`" + ` argument is true.",
									"locations": [
										"FIELD",
										"FRAGMENT_SPREAD",
										"INLINE_FRAGMENT"
									],
									"args": [
										{
											"name": "if",
											"description": "Skipped when true.",
											"type": {
												"kind": "NON_NULL",
												"ofType": {
													"kind": "SCALAR",
													"name": "Boolean"
												}
											}
										}
									]
								},
								{
									"name": "include",
									"description": "Directs the executor to include this field or fragment only when the ` + "`" + `if` + "`" + ` argument is true.",
									"locations": [
										"FIELD",
										"FRAGMENT_SPREAD",
										"INLINE_FRAGMENT"
									],
									"args": [
										{
											"name": "if",
											"description": "Included when true.",
											"type": {
												"kind": "NON_NULL",
												"ofType": {
													"kind": "SCALAR",
													"name": "Boolean"
												}
											}
										}
									]
								}
							]
						}
					}
			`,
		},
	})
}

func TestMutationOrder(t *testing.T) {
	graphql.RunTests(t, []*graphql.Test{
		{
			Schema: graphql.MustParseSchema(`
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
			Query: `
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
			ExpectedResult: `
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

	graphql.RunTests(t, []*graphql.Test{
		{
			Schema: schema,
			Query: `
				query($t: Time!) {
					a: addHour(time: $t)
					b: addHour
				}
			`,
			Variables: map[string]interface{}{
				"t": time.Date(2000, 2, 3, 4, 5, 6, 0, time.UTC),
			},
			ExpectedResult: `
				{
					"a": "2000-02-03T05:05:06Z",
					"b": "2001-02-03T05:05:06Z"
				}
			`,
		},
	})
}

type resolverWithUnexportedMethod struct{}

func (r *resolverWithUnexportedMethod) changeTheNumber(args *struct{ NewNumber int32 }) int32 {
	return args.NewNumber
}

func TestUnexportedMethod(t *testing.T) {
	_, err := graphql.ParseSchema(`
		schema {
			mutation: Mutation
		}

		type Mutation {
			changeTheNumber(newNumber: Int!): Int!
		}
	`, &resolverWithUnexportedMethod{})
	if err == nil {
		t.Error("error expected")
	}
}

type resolverWithUnexportedField struct{}

func (r *resolverWithUnexportedField) ChangeTheNumber(args *struct{ newNumber int32 }) int32 {
	return args.newNumber
}

func TestUnexportedField(t *testing.T) {
	_, err := graphql.ParseSchema(`
		schema {
			mutation: Mutation
		}

		type Mutation {
			changeTheNumber(newNumber: Int!): Int!
		}
	`, &resolverWithUnexportedField{})
	if err == nil {
		t.Error("error expected")
	}
}

type inputResolver struct{}

func (r *inputResolver) Int(args *struct{ Value int32 }) int32 {
	return args.Value
}

func (r *inputResolver) Float(args *struct{ Value float64 }) float64 {
	return args.Value
}

func (r *inputResolver) String(args *struct{ Value string }) string {
	return args.Value
}

func (r *inputResolver) Boolean(args *struct{ Value bool }) bool {
	return args.Value
}

func (r *inputResolver) Nullable(args *struct{ Value *int32 }) *int32 {
	return args.Value
}

func (r *inputResolver) List(args *struct{ Value []*struct{ V int32 } }) []int32 {
	l := make([]int32, len(args.Value))
	for i, entry := range args.Value {
		l[i] = entry.V
	}
	return l
}

func TestInput(t *testing.T) {
	coercionSchema := graphql.MustParseSchema(`
		schema {
			query: Query
		}

		type Query {
			int(value: Int!): Int!
			float(value: Float!): Float!
			string(value: String!): String!
			boolean(value: Boolean!): Boolean!
			nullable(value: Int): Int
			list(value: [Input!]!): [Int!]!
		}

		input Input {
			v: Int!
		}
	`, &inputResolver{})
	graphql.RunTests(t, []*graphql.Test{
		{
			Schema: coercionSchema,
			Query: `
				{
					int1: int(value: 42)
					int2: int(value: 42.0)
					float1: float(value: 42)
					float2: float(value: 42.5)
					string(value: "foo")
					boolean(value: true)
					nullable1: nullable(value: 42)
					nullable2: nullable(value: null)
					list(value: [{v: 41}, {v: 42}, {v: 43}])
				}
			`,
			ExpectedResult: `
				{
					"int1": 42,
					"int2": 42,
					"float1": 42,
					"float2": 42.5,
					"string": "foo",
					"boolean": true,
					"nullable1": 42,
					"nullable2": null,
					"list": [41, 42, 43]
				}
			`,
		},
	})
}
