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
		mutation: Mutation
	}
	# The query type, represents all of the entry points into our object graph
	type Query {
		hero(episode: Episode): Character
		reviews(episode: Episode!): [Review]
		search(text: String): [SearchResult]
		character(id: ID!): Character
		droid(id: ID!): Droid
		human(id: ID!): Human
		starship(id: ID!): Starship
	}
	# The mutation type, represents all updates we can make to our data
	type Mutation {
		createReview(episode: Episode, review: ReviewInput!): Review
	}
	# The episodes in the Star Wars trilogy
	enum Episode {
		# Star Wars Episode IV: A New Hope, released in 1977.
		NEWHOPE
		# Star Wars Episode V: The Empire Strikes Back, released in 1980.
		EMPIRE
		# Star Wars Episode VI: Return of the Jedi, released in 1983.
		JEDI
	}
	# A character from the Star Wars universe
	interface Character {
		# The ID of the character
		id: ID!
		# The name of the character
		name: String!
		# The friends of the character, or an empty list if they have none
		friends: [Character]
		# The friends of the character exposed as a connection with edges
		friendsConnection(first: Int, after: ID): FriendsConnection!
		# The movies this character appears in
		appearsIn: [Episode]!
	}
	# Units of height
	enum LengthUnit {
		# The standard unit around the world
		METER
		# Primarily used in the United States
		FOOT
	}
	# A humanoid creature from the Star Wars universe
	type Human implements Character {
		# The ID of the human
		id: ID!
		# What this human calls themselves
		name: String!
		# Height in the preferred unit, default is meters
		height(unit: LengthUnit = METER): Float
		# Mass in kilograms, or null if unknown
		mass: Float
		# This human's friends, or an empty list if they have none
		friends: [Character]
		# The friends of the human exposed as a connection with edges
		friendsConnection(first: Int, after: ID): FriendsConnection!
		# The movies this human appears in
		appearsIn: [Episode]!
		# A list of starships this person has piloted, or an empty list if none
		starships: [Starship]
	}
	# An autonomous mechanical character in the Star Wars universe
	type Droid implements Character {
		# The ID of the droid
		id: ID!
		# What others call this droid
		name: String!
		# This droid's friends, or an empty list if they have none
		friends: [Character]
		# The friends of the droid exposed as a connection with edges
		friendsConnection(first: Int, after: ID): FriendsConnection!
		# The movies this droid appears in
		appearsIn: [Episode]!
		# This droid's primary function
		primaryFunction: String
	}
	# A connection object for a character's friends
	type FriendsConnection {
		# The total number of friends
		totalCount: Int
		# The edges for each of the character's friends.
		edges: [FriendsEdge]
		# A list of the friends, as a convenience when edges are not needed.
		friends: [Character]
		# Information for paginating this connection
		pageInfo: PageInfo!
	}
	# An edge object for a character's friends
	type FriendsEdge {
		# A cursor used for pagination
		cursor: ID!
		# The character represented by this friendship edge
		node: Character
	}
	# Information for paginating this connection
	type PageInfo {
		startCursor: ID
		endCursor: ID
		hasNextPage: Boolean!
	}
	# Represents a review for a movie
	type Review {
		# The number of stars this review gave, 1-5
		stars: Int!
		# Comment about the movie
		commentary: String
	}
	# The input object sent when someone is creating a new review
	input ReviewInput {
		# 0-5 stars
		stars: Int!
		# Comment about the movie, optional
		commentary: String
	}
	type Starship {
		# The ID of the starship
		id: ID!
		# The name of the starship
		name: String!
		# Length of the starship, along the longest axis
		length(unit: LengthUnit = METER): Float
	}
	union SearchResult = Human | Droid | Starship
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
