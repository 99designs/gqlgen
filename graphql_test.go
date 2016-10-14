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

type Human struct {
	ID        string
	Name      string
	Friends   []string
	AppearsIn []string
	Height    float64
	Mass      int
	Starships []string
}

var humans = []*Human{
	{
		ID:        "1000",
		Name:      "Luke Skywalker",
		Friends:   []string{"1002", "1003", "2000", "2001"},
		AppearsIn: []string{"NEWHOPE", "EMPIRE", "JEDI"},
		Height:    1.72,
		Mass:      77,
		Starships: []string{"3001", "3003"},
	},
	{
		ID:        "1001",
		Name:      "Darth Vader",
		Friends:   []string{"1004"},
		AppearsIn: []string{"NEWHOPE", "EMPIRE", "JEDI"},
		Height:    2.02,
		Mass:      136,
		Starships: []string{"3002"},
	},
	{
		ID:        "1002",
		Name:      "Han Solo",
		Friends:   []string{"1000", "1003", "2001"},
		AppearsIn: []string{"NEWHOPE", "EMPIRE", "JEDI"},
		Height:    1.8,
		Mass:      80,
		Starships: []string{"3000", "3003"},
	},
	{
		ID:        "1003",
		Name:      "Leia Organa",
		Friends:   []string{"1000", "1002", "2000", "2001"},
		AppearsIn: []string{"NEWHOPE", "EMPIRE", "JEDI"},
		Height:    1.5,
		Mass:      49,
	},
	{
		ID:        "1004",
		Name:      "Wilhuff Tarkin",
		Friends:   []string{"1001"},
		AppearsIn: []string{"NEWHOPE"},
		Height:    1.8,
		Mass:      0,
	},
}

var humanData = make(map[string]*Human)

func init() {
	for _, h := range humans {
		humanData[h.ID] = h
	}
}

type Droid struct {
	ID              string
	Name            string
	Friends         []string
	AppearsIn       []string
	PrimaryFunction string
}

var droids = []*Droid{
	{
		ID:              "2000",
		Name:            "C-3PO",
		Friends:         []string{"1000", "1002", "1003", "2001"},
		AppearsIn:       []string{"NEWHOPE", "EMPIRE", "JEDI"},
		PrimaryFunction: "Protocol",
	},
	{
		ID:              "2001",
		Name:            "R2-D2",
		Friends:         []string{"1000", "1002", "1003"},
		AppearsIn:       []string{"NEWHOPE", "EMPIRE", "JEDI"},
		PrimaryFunction: "Astromech",
	},
}

var droidData = make(map[string]*Droid)

func init() {
	for _, d := range droids {
		droidData[d.ID] = d
	}
}

type Starship struct {
	ID     string
	Name   string
	Length float64
}

var starships = []*Starship{
	{
		ID:     "3000",
		Name:   "Millenium Falcon",
		Length: 34.37,
	},
	{
		ID:     "3001",
		Name:   "X-Wing",
		Length: 12.5,
	},
	{
		ID:     "3002",
		Name:   "TIE Advanced x1",
		Length: 9.2,
	},
	{
		ID:     "3003",
		Name:   "Imperial shuttle",
		Length: 20,
	},
}

var starshipData = make(map[string]*Starship)

func init() {
	for _, s := range starships {
		starshipData[s.ID] = s
	}
}

type starWarsResolver struct{}

func (r *starWarsResolver) Hero(args struct{ Episode string }) characterResolver {
	if args.Episode == "EMPIRE" {
		return &humanResolver{humanData["1000"]}
	}
	return &droidResolver{droidData["2001"]}
}

func (r *starWarsResolver) Human(args struct{ ID string }) *humanResolver {
	h := humanData[args.ID]
	if h == nil {
		return nil
	}
	return &humanResolver{h}
}

type characterResolver interface {
	ID() string
	Name() string
	Friends() []characterResolver
	AppearsIn() []string
}

type humanResolver struct {
	h *Human
}

func (r *humanResolver) ID() string {
	return r.h.ID
}

func (r *humanResolver) Name() string {
	return r.h.Name
}

func (r *humanResolver) Height(args struct{ Unit string }) float64 {
	switch args.Unit {
	case "METER":
		return r.h.Height
	case "FOOT":
		return r.h.Height * 3.28084
	default:
		panic("invalid unit")
	}
}

func (r *humanResolver) Friends() []characterResolver {
	return resolveCharacters(r.h.Friends)
}

func (r *humanResolver) AppearsIn() []string {
	return r.h.AppearsIn
}

type droidResolver struct {
	d *Droid
}

func (r *droidResolver) ID() string {
	return r.d.ID
}

func (r *droidResolver) Name() string {
	return r.d.Name
}

func (r *droidResolver) Friends() []characterResolver {
	return resolveCharacters(r.d.Friends)
}

func (r *droidResolver) AppearsIn() []string {
	return r.d.AppearsIn
}

func resolveCharacters(ids []string) []characterResolver {
	var characters []characterResolver
	for _, id := range ids {
		if h, ok := humanData[id]; ok {
			characters = append(characters, &humanResolver{h})
		}
		if d, ok := droidData[id]; ok {
			characters = append(characters, &droidResolver{d})
		}
	}
	return characters
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
		name:     "StarWarsBasic",
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
		name:     "StarWarsArguments1",
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

	{
		name:     "StarWarsArguments2",
		schema:   starWarsSchema,
		resolver: &starWarsResolver{},
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
		schema:   starWarsSchema,
		resolver: &starWarsResolver{},
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
		schema:   starWarsSchema,
		resolver: &starWarsResolver{},
		query: `
			{
				leftComparison: hero(episode: EMPIRE) {
					...comparisonFields
				}
				rightComparison: hero(episode: JEDI) {
					...comparisonFields
				}
			}
			
			fragment comparisonFields on Character {
				name
				appearsIn
				friends {
					name
				}
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
					]
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
