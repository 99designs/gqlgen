package graphql_test

import (
	"context"
	"testing"
	"time"

	"github.com/neelance/graphql-go"
	"github.com/neelance/graphql-go/example/starwars"
	"github.com/neelance/graphql-go/gqltesting"
)

type helloWorldResolver1 struct{}

func (r *helloWorldResolver1) Hello() string {
	return "Hello world!"
}

type helloWorldResolver2 struct{}

func (r *helloWorldResolver2) Hello(ctx context.Context) (string, error) {
	return "Hello world!", nil
}

type helloSnakeResolver1 struct{}

func (r *helloSnakeResolver1) HelloHTML() string {
	return "Hello snake!"
}

func (r *helloSnakeResolver1) SayHello(args struct{ FullName string }) string {
	return "Hello " + args.FullName + "!"
}

type helloSnakeResolver2 struct{}

func (r *helloSnakeResolver2) HelloHTML(ctx context.Context) (string, error) {
	return "Hello snake!", nil
}

func (r *helloSnakeResolver2) SayHello(ctx context.Context, args struct{ FullName string }) (string, error) {
	return "Hello " + args.FullName + "!", nil
}

type theNumberResolver struct {
	number int32
}

func (r *theNumberResolver) TheNumber() int32 {
	return r.number
}

func (r *theNumberResolver) ChangeTheNumber(args struct{ NewNumber int32 }) *theNumberResolver {
	r.number = args.NewNumber
	return r
}

type timeResolver struct{}

func (r *timeResolver) AddHour(args struct{ Time graphql.Time }) graphql.Time {
	return graphql.Time{Time: args.Time.Add(time.Hour)}
}

var starwarsSchema = graphql.MustParseSchema(starwars.Schema, &starwars.Resolver{})

func TestHelloWorld(t *testing.T) {
	gqltesting.RunTests(t, []*gqltesting.Test{
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

func TestHelloSnake(t *testing.T) {
	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Schema: graphql.MustParseSchema(`
				schema {
					query: Query
				}

				type Query {
					hello_html: String!
				}
			`, &helloSnakeResolver1{}),
			Query: `
				{
					hello_html
				}
			`,
			ExpectedResult: `
				{
					"hello_html": "Hello snake!"
				}
			`,
		},

		{
			Schema: graphql.MustParseSchema(`
				schema {
					query: Query
				}

				type Query {
					hello_html: String!
				}
			`, &helloSnakeResolver2{}),
			Query: `
				{
					hello_html
				}
			`,
			ExpectedResult: `
				{
					"hello_html": "Hello snake!"
				}
			`,
		},
	})
}

func TestHelloSnakeArguments(t *testing.T) {
	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Schema: graphql.MustParseSchema(`
				schema {
					query: Query
				}

				type Query {
					say_hello(full_name: String!): String!
				}
			`, &helloSnakeResolver1{}),
			Query: `
				{
					say_hello(full_name: "Rob Pike")
				}
			`,
			ExpectedResult: `
				{
					"say_hello": "Hello Rob Pike!"
				}
			`,
		},

		{
			Schema: graphql.MustParseSchema(`
				schema {
					query: Query
				}

				type Query {
					say_hello(full_name: String!): String!
				}
			`, &helloSnakeResolver2{}),
			Query: `
				{
					say_hello(full_name: "Rob Pike")
				}
			`,
			ExpectedResult: `
				{
					"say_hello": "Hello Rob Pike!"
				}
			`,
		},
	})
}

func TestBasic(t *testing.T) {
	gqltesting.RunTests(t, []*gqltesting.Test{
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
	gqltesting.RunTests(t, []*gqltesting.Test{
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
	gqltesting.RunTests(t, []*gqltesting.Test{
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
	gqltesting.RunTests(t, []*gqltesting.Test{
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
	gqltesting.RunTests(t, []*gqltesting.Test{
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
	gqltesting.RunTests(t, []*gqltesting.Test{
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
	gqltesting.RunTests(t, []*gqltesting.Test{
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

type testDeprecatedDirectiveResolver struct{}

func (r *testDeprecatedDirectiveResolver) A() int32 {
	return 0
}

func (r *testDeprecatedDirectiveResolver) B() int32 {
	return 0
}

func (r *testDeprecatedDirectiveResolver) C() int32 {
	return 0
}

func TestDeprecatedDirective(t *testing.T) {
	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Schema: graphql.MustParseSchema(`
				schema {
					query: Query
				}

				type Query {
					a: Int!
					b: Int! @deprecated
					c: Int! @deprecated(reason: "We don't like it")
				}
			`, &testDeprecatedDirectiveResolver{}),
			Query: `
				{
					__type(name: "Query") {
						fields {
							name
						}
						allFields: fields(includeDeprecated: true) {
							name
							isDeprecated
							deprecationReason
						}
					}
				}
			`,
			ExpectedResult: `
				{
					"__type": {
						"fields": [
							{ "name": "a" }
						],
						"allFields": [
							{ "name": "a", "isDeprecated": false, "deprecationReason": null },
							{ "name": "b", "isDeprecated": true, "deprecationReason": "No longer supported" },
							{ "name": "c", "isDeprecated": true, "deprecationReason": "We don't like it" }
						]
					}
				}
			`,
		},
		{
			Schema: graphql.MustParseSchema(`
				schema {
					query: Query
				}

				type Query {
				}

				enum Test {
					A
					B @deprecated
					C @deprecated(reason: "We don't like it")
				}
			`, &testDeprecatedDirectiveResolver{}),
			Query: `
				{
					__type(name: "Test") {
						enumValues {
							name
						}
						allEnumValues: enumValues(includeDeprecated: true) {
							name
							isDeprecated
							deprecationReason
						}
					}
				}
			`,
			ExpectedResult: `
				{
					"__type": {
						"enumValues": [
							{ "name": "A" }
						],
						"allEnumValues": [
							{ "name": "A", "isDeprecated": false, "deprecationReason": null },
							{ "name": "B", "isDeprecated": true, "deprecationReason": "No longer supported" },
							{ "name": "C", "isDeprecated": true, "deprecationReason": "We don't like it" }
						]
					}
				}
			`,
		},
	})
}

func TestInlineFragments(t *testing.T) {
	gqltesting.RunTests(t, []*gqltesting.Test{
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
	gqltesting.RunTests(t, []*gqltesting.Test{
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
	gqltesting.RunTests(t, []*gqltesting.Test{
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
	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Schema: starwarsSchema,
			Query: `
				{
					reviews(episode: JEDI) {
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
					reviews(episode: JEDI) {
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
	gqltesting.RunTests(t, []*gqltesting.Test{
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
									"name": "deprecated",
									"description": "Marks an element of a GraphQL schema as no longer supported.",
									"locations": [
										"FIELD_DEFINITION",
										"ENUM_VALUE"
									],
									"args": [
										{
											"name": "reason",
											"description": "Explains why this element was deprecated, usually also including a suggestion\nfor how to access supported similar data. Formatted in\n[Markdown](https://daringfireball.net/projects/markdown/).",
											"type": {
												"kind": "SCALAR",
												"ofType": null
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
								},
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
								}
							]
						}
					}
			`,
		},
	})
}

func TestMutationOrder(t *testing.T) {
	gqltesting.RunTests(t, []*gqltesting.Test{
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
	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Schema: graphql.MustParseSchema(`
				schema {
					query: Query
				}

				type Query {
					addHour(time: Time = "2001-02-03T04:05:06Z"): Time!
				}

				scalar Time
			`, &timeResolver{}),
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

func (r *resolverWithUnexportedMethod) changeTheNumber(args struct{ NewNumber int32 }) int32 {
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

func (r *resolverWithUnexportedField) ChangeTheNumber(args struct{ newNumber int32 }) int32 {
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

func (r *inputResolver) Int(args struct{ Value int32 }) int32 {
	return args.Value
}

func (r *inputResolver) Float(args struct{ Value float64 }) float64 {
	return args.Value
}

func (r *inputResolver) String(args struct{ Value string }) string {
	return args.Value
}

func (r *inputResolver) Boolean(args struct{ Value bool }) bool {
	return args.Value
}

func (r *inputResolver) Nullable(args struct{ Value *int32 }) *int32 {
	return args.Value
}

func (r *inputResolver) List(args struct{ Value []*struct{ V int32 } }) []int32 {
	l := make([]int32, len(args.Value))
	for i, entry := range args.Value {
		l[i] = entry.V
	}
	return l
}

func (r *inputResolver) NullableList(args struct{ Value *[]*struct{ V int32 } }) *[]*int32 {
	if args.Value == nil {
		return nil
	}
	l := make([]*int32, len(*args.Value))
	for i, entry := range *args.Value {
		if entry != nil {
			l[i] = &entry.V
		}
	}
	return &l
}

func (r *inputResolver) Enum(args struct{ Value string }) string {
	return args.Value
}

func (r *inputResolver) NullableEnum(args struct{ Value *string }) *string {
	return args.Value
}

type recursive struct {
	Next *recursive
}

func (r *inputResolver) Recursive(args struct{ Value *recursive }) int32 {
	n := int32(0)
	v := args.Value
	for v != nil {
		v = v.Next
		n++
	}
	return n
}

func (r *inputResolver) ID(args struct{ Value graphql.ID }) graphql.ID {
	return args.Value
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
			nullableList(value: [Input]): [Int]
			enum(value: Enum!): Enum!
			nullableEnum(value: Enum): Enum
			recursive(value: RecursiveInput!): Int!
			id(value: ID!): ID!
		}

		input Input {
			v: Int!
		}

		input RecursiveInput {
			next: RecursiveInput
		}

		enum Enum {
			Option1
			Option2
		}
	`, &inputResolver{})
	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Schema: coercionSchema,
			Query: `
				{
					int(value: 42)
					float1: float(value: 42)
					float2: float(value: 42.5)
					string(value: "foo")
					boolean(value: true)
					nullable1: nullable(value: 42)
					nullable2: nullable(value: null)
					list1: list(value: [{v: 41}, {v: 42}, {v: 43}])
					list2: list(value: {v: 42})
					nullableList1: nullableList(value: [{v: 41}, null, {v: 43}])
					nullableList2: nullableList(value: null)
					enum(value: Option2)
					nullableEnum1: nullableEnum(value: Option2)
					nullableEnum2: nullableEnum(value: null)
					recursive(value: {next: {next: {}}})
					intID: id(value: 1234)
					strID: id(value: "1234")
				}
			`,
			ExpectedResult: `
				{
					"int": 42,
					"float1": 42,
					"float2": 42.5,
					"string": "foo",
					"boolean": true,
					"nullable1": 42,
					"nullable2": null,
					"list1": [41, 42, 43],
					"list2": [42],
					"nullableList1": [41, null, 43],
					"nullableList2": null,
					"enum": "Option2",
					"nullableEnum1": "Option2",
					"nullableEnum2": null,
					"recursive": 3,
					"intID": "1234",
					"strID": "1234"
				}
			`,
		},
	})
}

func TestComposedFragments(t *testing.T) {
	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Schema: starwarsSchema,
			Query: `
				{
					composed: hero(episode: EMPIRE) {
						name
						...friendsNames
						...friendsIds
					}
				}

				fragment friendsNames on Character {
					name
					friends {
						name
					}
				}

				fragment friendsIds on Character {
					name
					friends {
						id
					}
				}
			`,
			ExpectedResult: `
				{
					"composed": {
						"name": "Luke Skywalker",
						"friends": [
							{
								"id": "1002",
								"name": "Han Solo"
							},
							{
								"id": "1003",
								"name": "Leia Organa"
							},
							{
								"id": "2000",
								"name": "C-3PO"
							},
							{
								"id": "2001",
								"name": "R2-D2"
							}
						]
					}
				}
			`,
		},
	})
}
