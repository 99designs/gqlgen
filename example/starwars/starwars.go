// Package starwars provides a example schema and resolver based on Star Wars characters.
//
// Source: https://github.com/graphql/graphql.github.io/blob/source/site/_core/swapiSchema.js
package starwars

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

var Schema = `
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
		appearsIn: [Episode!]!
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
		height(unit: LengthUnit = METER): Float!
		# Mass in kilograms, or null if unknown
		mass: Float
		# This human's friends, or an empty list if they have none
		friends: [Character]
		# The friends of the human exposed as a connection with edges
		friendsConnection(first: Int, after: ID): FriendsConnection!
		# The movies this human appears in
		appearsIn: [Episode!]!
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
		appearsIn: [Episode!]!
		# This droid's primary function
		primaryFunction: String
	}
	# A connection object for a character's friends
	type FriendsConnection {
		# The total number of friends
		totalCount: Int!
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
		length(unit: LengthUnit = METER): Float!
	}
	union SearchResult = Human | Droid | Starship
`

type human struct {
	ID        string
	Name      string
	Friends   []string
	AppearsIn []string
	Height    float64
	Mass      int
	Starships []string
}

var humans = []*human{
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

var humanData = make(map[string]*human)

func init() {
	for _, h := range humans {
		humanData[h.ID] = h
	}
}

type droid struct {
	ID              string
	Name            string
	Friends         []string
	AppearsIn       []string
	PrimaryFunction string
}

var droids = []*droid{
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

var droidData = make(map[string]*droid)

func init() {
	for _, d := range droids {
		droidData[d.ID] = d
	}
}

type starship struct {
	ID     string
	Name   string
	Length float64
}

var starships = []*starship{
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

var starshipData = make(map[string]*starship)

func init() {
	for _, s := range starships {
		starshipData[s.ID] = s
	}
}

type Resolver struct{}

func (r *Resolver) Hero(args struct{ Episode string }) characterResolver {
	if args.Episode == "EMPIRE" {
		return &humanResolver{h: humanData["1000"]}
	}
	return &droidResolver{d: droidData["2001"]}
}

func (r *Resolver) Reviews(args struct{ Episode string }) *[]*reviewResolver {
	panic("TODO")
}

func (r *Resolver) Search(args struct{ Text string }) *[]searchResultResolver {
	var l []searchResultResolver
	for _, h := range humans {
		if strings.Contains(h.Name, args.Text) {
			l = append(l, &humanResolver{h: h})
		}
	}
	for _, d := range droids {
		if strings.Contains(d.Name, args.Text) {
			l = append(l, &droidResolver{d: d})
		}
	}
	for _, s := range starships {
		if strings.Contains(s.Name, args.Text) {
			l = append(l, &starshipResolver{s: s})
		}
	}
	return &l
}

func (r *Resolver) Character(args struct{ ID string }) characterResolver {
	if h := humanData[args.ID]; h != nil {
		return &humanResolver{h: h}
	}
	if d := droidData[args.ID]; d != nil {
		return &droidResolver{d: d}
	}
	return nil
}

func (r *Resolver) Human(args struct{ ID string }) *humanResolver {
	if h := humanData[args.ID]; h != nil {
		return &humanResolver{h: h}
	}
	return nil
}

func (r *Resolver) Droid(args struct{ ID string }) *droidResolver {
	if d := droidData[args.ID]; d != nil {
		return &droidResolver{d: d}
	}
	return nil
}

func (r *Resolver) Starship(args struct{ ID string }) *starshipResolver {
	if s := starshipData[args.ID]; s != nil {
		return &starshipResolver{s: s}
	}
	return nil
}

type friendsConenctionArgs struct {
	First int32
	After string
}

type characterResolver interface {
	ID() string
	Name() string
	Friends() *[]characterResolver
	FriendsConnection(friendsConenctionArgs) (*friendsConnectionResolver, error)
	AppearsIn() []string
	ToHuman() (*humanResolver, bool)
	ToDroid() (*droidResolver, bool)
}

type characterBase struct{}

func (r characterBase) ToHuman() (*humanResolver, bool) {
	return nil, false
}

func (r characterBase) ToDroid() (*droidResolver, bool) {
	return nil, false
}

func (r characterBase) ToStarship() (*starshipResolver, bool) {
	return nil, false
}

type humanResolver struct {
	characterBase
	h *human
}

func (r *humanResolver) ID() string {
	return r.h.ID
}

func (r *humanResolver) Name() string {
	return r.h.Name
}

func (r *humanResolver) Height(args struct{ Unit string }) float64 {
	return convertLength(r.h.Height, args.Unit)
}

func (r *humanResolver) Mass() *float64 {
	if r.h.Mass == 0 {
		return nil
	}
	f := float64(r.h.Mass)
	return &f
}

func (r *humanResolver) Friends() *[]characterResolver {
	return resolveCharacters(r.h.Friends)
}

func (r *humanResolver) FriendsConnection(args friendsConenctionArgs) (*friendsConnectionResolver, error) {
	return newFriendsConnectionResolver(r.h.Friends, args)
}

func (r *humanResolver) AppearsIn() []string {
	return r.h.AppearsIn
}

func (r *humanResolver) Starships() *[]*starshipResolver {
	l := make([]*starshipResolver, len(r.h.Starships))
	for i, id := range r.h.Starships {
		l[i] = &starshipResolver{s: starshipData[id]}
	}
	return &l
}

func (r *humanResolver) ToHuman() (*humanResolver, bool) {
	return r, true
}

type droidResolver struct {
	characterBase
	d *droid
}

func (r *droidResolver) ID() string {
	return r.d.ID
}

func (r *droidResolver) Name() string {
	return r.d.Name
}

func (r *droidResolver) Friends() *[]characterResolver {
	return resolveCharacters(r.d.Friends)
}

func (r *droidResolver) FriendsConnection(args friendsConenctionArgs) (*friendsConnectionResolver, error) {
	return newFriendsConnectionResolver(r.d.Friends, args)
}

func (r *droidResolver) AppearsIn() []string {
	return r.d.AppearsIn
}

func (r *droidResolver) PrimaryFunction() *string {
	if r.d.PrimaryFunction == "" {
		return nil
	}
	return &r.d.PrimaryFunction
}

func (r *droidResolver) ToDroid() (*droidResolver, bool) {
	return r, true
}

type starshipResolver struct {
	characterBase
	s *starship
}

func (r *starshipResolver) ID() string {
	return r.s.ID
}

func (r *starshipResolver) Name() string {
	return r.s.Name
}

func (r *starshipResolver) Length(args struct{ Unit string }) float64 {
	return convertLength(r.s.Length, args.Unit)
}

func (r *starshipResolver) ToStarship() (*starshipResolver, bool) {
	return r, true
}

type searchResultResolver interface {
	ToHuman() (*humanResolver, bool)
	ToDroid() (*droidResolver, bool)
	ToStarship() (*starshipResolver, bool)
}

func convertLength(meters float64, unit string) float64 {
	switch unit {
	case "METER":
		return meters
	case "FOOT":
		return meters * 3.28084
	default:
		panic("invalid unit")
	}
}

func resolveCharacters(ids []string) *[]characterResolver {
	var characters []characterResolver
	for _, id := range ids {
		if c := resolveCharacter(id); c != nil {
			characters = append(characters, c)
		}
	}
	return &characters
}

func resolveCharacter(id string) characterResolver {
	if h, ok := humanData[id]; ok {
		return &humanResolver{h: h}
	}
	if d, ok := droidData[id]; ok {
		return &droidResolver{d: d}
	}
	return nil
}

type reviewResolver struct {
}

func (r *reviewResolver) Stars() int32 {
	panic("TODO")
}

func (r *reviewResolver) Commentary() *string {
	panic("TODO")
}

type friendsConnectionResolver struct {
	ids  []string
	from int
	to   int
}

func newFriendsConnectionResolver(ids []string, args friendsConenctionArgs) (*friendsConnectionResolver, error) {
	from := 0
	if args.After != "" {
		b, err := base64.StdEncoding.DecodeString(args.After)
		if err != nil {
			return nil, err
		}
		i, err := strconv.Atoi(strings.TrimPrefix(string(b), "cursor"))
		if err != nil {
			return nil, err
		}
		from = i
	}

	to := len(ids)
	if args.First != 0 {
		to = from + int(args.First)
	}
	if to > len(ids)-from {
		to = len(ids) - from
	}

	return &friendsConnectionResolver{
		ids:  ids,
		from: from,
		to:   to,
	}, nil
}

func (r *friendsConnectionResolver) TotalCount() int32 {
	return int32(len(r.ids))
}

func (r *friendsConnectionResolver) Edges() *[]*friendsEdgeResolver {
	l := make([]*friendsEdgeResolver, r.to-r.from)
	for i := range l {
		l[i] = &friendsEdgeResolver{
			cursor: encodeCursor(r.from + i),
			id:     r.ids[r.from+i],
		}
	}
	return &l
}

func (r *friendsConnectionResolver) Friends() *[]characterResolver {
	return resolveCharacters(r.ids[r.from:r.to])
}

func (r *friendsConnectionResolver) PageInfo() *pageInfoResolver {
	return &pageInfoResolver{
		startCursor: encodeCursor(r.from),
		endCursor:   encodeCursor(r.to - 1),
		hasNextPage: r.to < len(r.ids),
	}
}

func encodeCursor(i int) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("cursor%d", i+1)))
}

type friendsEdgeResolver struct {
	cursor string
	id     string
}

func (r *friendsEdgeResolver) Cursor() string {
	return r.cursor
}

func (r *friendsEdgeResolver) Node() characterResolver {
	return resolveCharacter(r.id)
}

type pageInfoResolver struct {
	startCursor string
	endCursor   string
	hasNextPage bool
}

func (r *pageInfoResolver) StartCursor() *string {
	return &r.startCursor
}

func (r *pageInfoResolver) EndCursor() *string {
	return &r.endCursor
}

func (r *pageInfoResolver) HasNextPage() bool {
	return r.hasNextPage
}
