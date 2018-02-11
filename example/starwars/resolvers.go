//go:generate gorunpkg github.com/vektah/gqlgen -out generated.go

package starwars

import (
	"context"
	"strings"
	"time"
)

type Resolver struct {
	humans    map[string]Human
	droid     map[string]Droid
	starships map[string]Starship
	reviews   map[string][]Review
}

func (r *Resolver) resolveCharacters(ctx context.Context, ids []string) ([]Character, error) {
	var result []Character
	for _, id := range ids {
		char, err := r.Query_character(ctx, id)
		if err != nil {
			return nil, err
		}
		result = append(result, char)
	}
	return result, nil
}

func (r *Resolver) Human_friends(ctx context.Context, it *Human) ([]Character, error) {
	return r.resolveCharacters(ctx, it.FriendIds)
}

func (r *Resolver) Human_friendsConnection(ctx context.Context, it *Human, first *int, after *string) (FriendsConnection, error) {
	return r.resolveFriendConnection(ctx, it.FriendIds, first, after)
}

func (r *Resolver) Human_starships(ctx context.Context, it *Human) ([]Starship, error) {
	var result []Starship
	for _, id := range it.StarshipIds {
		char, err := r.Query_starship(ctx, id)
		if err != nil {
			return nil, err
		}
		if char != nil {
			result = append(result, *char)
		}
	}
	return result, nil
}

func (r *Resolver) Droid_friends(ctx context.Context, it *Droid) ([]Character, error) {
	return r.resolveCharacters(ctx, it.FriendIds)
}

func (r *Resolver) Droid_friendsConnection(ctx context.Context, it *Droid, first *int, after *string) (FriendsConnection, error) {
	return r.resolveFriendConnection(ctx, it.FriendIds, first, after)
}

func (r *Resolver) FriendsConnection_edges(ctx context.Context, it *FriendsConnection) ([]FriendsEdge, error) {
	friends, err := r.resolveCharacters(ctx, it.ids)
	if err != nil {
		return nil, err
	}

	edges := make([]FriendsEdge, it.to-it.from)
	for i := range edges {
		edges[i] = FriendsEdge{
			Cursor: encodeCursor(it.from + i),
			Node:   friends[i],
		}
	}
	return edges, nil
}

// A list of the friends, as a convenience when edges are not needed.
func (r *Resolver) FriendsConnection_friends(ctx context.Context, it *FriendsConnection) ([]Character, error) {
	return r.resolveCharacters(ctx, it.ids)
}

func (r *Resolver) Mutation_createReview(ctx context.Context, episode string, review Review) (*Review, error) {
	review.Time = time.Now()
	time.Sleep(1 * time.Second)
	r.reviews[episode] = append(r.reviews[episode], review)
	return &review, nil
}

func (r *Resolver) Query_hero(ctx context.Context, episode *string) (Character, error) {
	if episode != nil && *episode == "EMPIRE" {
		return r.humans["1000"], nil
	}
	return r.droid["2001"], nil
}

func (r *Resolver) Query_reviews(ctx context.Context, episode string, since *time.Time) ([]Review, error) {
	if since == nil {
		return r.reviews[episode], nil
	}

	var filtered []Review
	for _, rev := range r.reviews[episode] {
		if rev.Time.After(*since) {
			filtered = append(filtered, rev)
		}
	}
	return filtered, nil
}

func (r *Resolver) Query_search(ctx context.Context, text string) ([]SearchResult, error) {
	var l []SearchResult
	for _, h := range r.humans {
		if strings.Contains(h.Name, text) {
			l = append(l, h)
		}
	}
	for _, d := range r.droid {
		if strings.Contains(d.Name, text) {
			l = append(l, d)
		}
	}
	for _, s := range r.starships {
		if strings.Contains(s.Name, text) {
			l = append(l, s)
		}
	}
	return l, nil
}

func (r *Resolver) Query_character(ctx context.Context, id string) (Character, error) {
	if h, ok := r.humans[id]; ok {
		return &h, nil
	}
	if d, ok := r.droid[id]; ok {
		return &d, nil
	}
	return nil, nil
}
func (r *Resolver) Query_droid(ctx context.Context, id string) (*Droid, error) {
	if d, ok := r.droid[id]; ok {
		return &d, nil
	}
	return nil, nil
}
func (r *Resolver) Query_human(ctx context.Context, id string) (*Human, error) {
	if h, ok := r.humans[id]; ok {
		return &h, nil
	}
	return nil, nil
}
func (r *Resolver) Query_starship(ctx context.Context, id string) (*Starship, error) {
	if s, ok := r.starships[id]; ok {
		return &s, nil
	}
	return nil, nil
}

func NewResolver() *Resolver {
	r := Resolver{}
	r.humans = map[string]Human{
		"1000": {
			ID:           "1000",
			Name:         "Luke Skywalker",
			FriendIds:    []string{"1002", "1003", "2000", "2001"},
			AppearsIn:    []string{"NEWHOPE", "EMPIRE", "JEDI"},
			heightMeters: 1.72,
			Mass:         77,
			StarshipIds:  []string{"3001", "3003"},
		},
		"1001": {
			ID:           "1001",
			Name:         "Darth Vader",
			FriendIds:    []string{"1004"},
			AppearsIn:    []string{"NEWHOPE", "EMPIRE", "JEDI"},
			heightMeters: 2.02,
			Mass:         136,
			StarshipIds:  []string{"3002"},
		},
		"1002": {
			ID:           "1002",
			Name:         "Han Solo",
			FriendIds:    []string{"1000", "1003", "2001"},
			AppearsIn:    []string{"NEWHOPE", "EMPIRE", "JEDI"},
			heightMeters: 1.8,
			Mass:         80,
			StarshipIds:  []string{"3000", "3003"},
		},
		"1003": {
			ID:           "1003",
			Name:         "Leia Organa",
			FriendIds:    []string{"1000", "1002", "2000", "2001"},
			AppearsIn:    []string{"NEWHOPE", "EMPIRE", "JEDI"},
			heightMeters: 1.5,
			Mass:         49,
		},
		"1004": {
			ID:           "1004",
			Name:         "Wilhuff Tarkin",
			FriendIds:    []string{"1001"},
			AppearsIn:    []string{"NEWHOPE"},
			heightMeters: 1.8,
			Mass:         0,
		},
	}

	r.droid = map[string]Droid{
		"2000": {
			ID:              "2000",
			Name:            "C-3PO",
			FriendIds:       []string{"1000", "1002", "1003", "2001"},
			AppearsIn:       []string{"NEWHOPE", "EMPIRE", "JEDI"},
			PrimaryFunction: "Protocol",
		},
		"2001": {
			ID:              "2001",
			Name:            "R2-D2",
			FriendIds:       []string{"1000", "1002", "1003"},
			AppearsIn:       []string{"NEWHOPE", "EMPIRE", "JEDI"},
			PrimaryFunction: "Astromech",
		},
	}

	r.starships = map[string]Starship{
		"3000": {
			ID:   "3000",
			Name: "Millennium Falcon",
			History: [][2]int{
				{1, 2},
				{4, 5},
				{1, 2},
				{3, 2},
			},
			lengthMeters: 34.37,
		},
		"3001": {
			ID:   "3001",
			Name: "X-Wing",
			History: [][2]int{
				{6, 4},
				{3, 2},
				{2, 3},
				{5, 1},
			},
			lengthMeters: 12.5,
		},
		"3002": {
			ID:   "3002",
			Name: "TIE Advanced x1",
			History: [][2]int{
				{3, 2},
				{7, 2},
				{6, 4},
				{3, 2},
			},
			lengthMeters: 9.2,
		},
		"3003": {
			ID:   "3003",
			Name: "Imperial shuttle",
			History: [][2]int{
				{1, 7},
				{3, 5},
				{5, 3},
				{7, 1},
			},
			lengthMeters: 20,
		},
	}

	r.reviews = map[string][]Review{}

	return &r
}
