//go:generate gorunpkg github.com/vektah/gqlgen

package starwars

import (
	"context"
	"errors"
	"strings"
	"time"
)

type Resolver struct {
	humans    map[string]Human
	droid     map[string]Droid
	starships map[string]Starship
	reviews   map[Episode][]Review
}

func (r *Resolver) Droid() DroidResolver {
	return &droidResolver{r}
}

func (r *Resolver) FriendsConnection() FriendsConnectionResolver {
	return &friendsConnectionResolver{r}
}

func (r *Resolver) Human() HumanResolver {
	return &humanResolver{r}
}

func (r *Resolver) Mutation() MutationResolver {
	return &mutationResolver{r}
}

func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}

func (r *Resolver) Starship() StarshipResolver {
	return &starshipResolver{r}
}

func (r *Resolver) resolveCharacters(ctx context.Context, ids []string) ([]Character, error) {
	var result []Character
	for _, id := range ids {
		char, err := r.Query().Character(ctx, id)
		if err != nil {
			return nil, err
		}
		result = append(result, char)
	}
	return result, nil
}

type droidResolver struct{ *Resolver }

func (r *droidResolver) Friends(ctx context.Context, obj *Droid) ([]Character, error) {
	return r.resolveCharacters(ctx, obj.FriendIds)
}

func (r *droidResolver) FriendsConnection(ctx context.Context, obj *Droid, first *int, after *string) (FriendsConnection, error) {
	return r.resolveFriendConnection(ctx, obj.FriendIds, first, after)
}

type friendsConnectionResolver struct{ *Resolver }

func (r *friendsConnectionResolver) Edges(ctx context.Context, obj *FriendsConnection) ([]FriendsEdge, error) {
	friends, err := r.resolveCharacters(ctx, obj.ids)
	if err != nil {
		return nil, err
	}

	edges := make([]FriendsEdge, obj.to-obj.from)
	for i := range edges {
		edges[i] = FriendsEdge{
			Cursor: encodeCursor(obj.from + i),
			Node:   friends[i],
		}
	}
	return edges, nil
}

func (r *friendsConnectionResolver) Friends(ctx context.Context, obj *FriendsConnection) ([]Character, error) {
	return r.resolveCharacters(ctx, obj.ids)
}

type humanResolver struct{ *Resolver }

func (r *humanResolver) Friends(ctx context.Context, obj *Human) ([]Character, error) {
	return r.resolveCharacters(ctx, obj.FriendIds)
}

func (r *humanResolver) FriendsConnection(ctx context.Context, obj *Human, first *int, after *string) (FriendsConnection, error) {
	return r.resolveFriendConnection(ctx, obj.FriendIds, first, after)
}

func (r *humanResolver) Starships(ctx context.Context, obj *Human) ([]Starship, error) {
	var result []Starship
	for _, id := range obj.StarshipIds {
		char, err := r.Query().Starship(ctx, id)
		if err != nil {
			return nil, err
		}
		if char != nil {
			result = append(result, *char)
		}
	}
	return result, nil
}

type mutationResolver struct{ *Resolver }

func (r *mutationResolver) CreateReview(ctx context.Context, episode Episode, review Review) (*Review, error) {
	review.Time = time.Now()
	time.Sleep(1 * time.Second)
	r.reviews[episode] = append(r.reviews[episode], review)
	return &review, nil
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Hero(ctx context.Context, episode Episode) (Character, error) {
	if episode == EpisodeEmpire {
		return r.humans["1000"], nil
	}
	return r.droid["2001"], nil
}

func (r *queryResolver) Reviews(ctx context.Context, episode Episode, since *time.Time) ([]Review, error) {
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

func (r *queryResolver) Search(ctx context.Context, text string) ([]SearchResult, error) {
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

func (r *queryResolver) Character(ctx context.Context, id string) (Character, error) {
	if h, ok := r.humans[id]; ok {
		return &h, nil
	}
	if d, ok := r.droid[id]; ok {
		return &d, nil
	}
	return nil, nil
}

func (r *queryResolver) Droid(ctx context.Context, id string) (*Droid, error) {
	if d, ok := r.droid[id]; ok {
		return &d, nil
	}
	return nil, nil
}

func (r *queryResolver) Human(ctx context.Context, id string) (*Human, error) {
	if h, ok := r.humans[id]; ok {
		return &h, nil
	}
	return nil, nil
}

func (r *queryResolver) Starship(ctx context.Context, id string) (*Starship, error) {
	if s, ok := r.starships[id]; ok {
		return &s, nil
	}
	return nil, nil
}

type starshipResolver struct{ *Resolver }

func (r *starshipResolver) Length(ctx context.Context, obj *Starship, unit LengthUnit) (float64, error) {
	switch unit {
	case LengthUnitMeter, "":
		return obj.Length, nil
	case LengthUnitFoot:
		return obj.Length * 3.28084, nil
	default:
		return 0, errors.New("invalid unit")
	}
}

func NewResolver() Config {
	r := Resolver{}
	r.humans = map[string]Human{
		"1000": {
			CharacterFields: CharacterFields{
				ID:        "1000",
				Name:      "Luke Skywalker",
				FriendIds: []string{"1002", "1003", "2000", "2001"},
				AppearsIn: []Episode{EpisodeNewhope, EpisodeEmpire, EpisodeJedi},
			},
			heightMeters: 1.72,
			Mass:         77,
			StarshipIds:  []string{"3001", "3003"},
		},
		"1001": {
			CharacterFields: CharacterFields{
				ID:        "1001",
				Name:      "Darth Vader",
				FriendIds: []string{"1004"},
				AppearsIn: []Episode{EpisodeNewhope, EpisodeEmpire, EpisodeJedi},
			},
			heightMeters: 2.02,
			Mass:         136,
			StarshipIds:  []string{"3002"},
		},
		"1002": {
			CharacterFields: CharacterFields{
				ID:        "1002",
				Name:      "Han Solo",
				FriendIds: []string{"1000", "1003", "2001"},
				AppearsIn: []Episode{EpisodeNewhope, EpisodeEmpire, EpisodeJedi},
			},
			heightMeters: 1.8,
			Mass:         80,
			StarshipIds:  []string{"3000", "3003"},
		},
		"1003": {
			CharacterFields: CharacterFields{
				ID:        "1003",
				Name:      "Leia Organa",
				FriendIds: []string{"1000", "1002", "2000", "2001"},
				AppearsIn: []Episode{EpisodeNewhope, EpisodeEmpire, EpisodeJedi},
			},
			heightMeters: 1.5,
			Mass:         49,
		},
		"1004": {
			CharacterFields: CharacterFields{
				ID:        "1004",
				Name:      "Wilhuff Tarkin",
				FriendIds: []string{"1001"},
				AppearsIn: []Episode{EpisodeNewhope},
			},
			heightMeters: 1.8,
			Mass:         0,
		},
	}

	r.droid = map[string]Droid{
		"2000": {
			CharacterFields: CharacterFields{
				ID:        "2000",
				Name:      "C-3PO",
				FriendIds: []string{"1000", "1002", "1003", "2001"},
				AppearsIn: []Episode{EpisodeNewhope, EpisodeEmpire, EpisodeJedi},
			},
			PrimaryFunction: "Protocol",
		},
		"2001": {
			CharacterFields: CharacterFields{
				ID:        "2001",
				Name:      "R2-D2",
				FriendIds: []string{"1000", "1002", "1003"},
				AppearsIn: []Episode{EpisodeNewhope, EpisodeEmpire, EpisodeJedi},
			},
			PrimaryFunction: "Astromech",
		},
	}

	r.starships = map[string]Starship{
		"3000": {
			ID:   "3000",
			Name: "Millennium Falcon",
			History: [][]int{
				{1, 2},
				{4, 5},
				{1, 2},
				{3, 2},
			},
			Length: 34.37,
		},
		"3001": {
			ID:   "3001",
			Name: "X-Wing",
			History: [][]int{
				{6, 4},
				{3, 2},
				{2, 3},
				{5, 1},
			},
			Length: 12.5,
		},
		"3002": {
			ID:   "3002",
			Name: "TIE Advanced x1",
			History: [][]int{
				{3, 2},
				{7, 2},
				{6, 4},
				{3, 2},
			},
			Length: 9.2,
		},
		"3003": {
			ID:   "3003",
			Name: "Imperial shuttle",
			History: [][]int{
				{1, 7},
				{3, 5},
				{5, 3},
				{7, 1},
			},
			Length: 20,
		},
	}

	r.reviews = map[Episode][]Review{}

	return Config{
		Resolvers: &r,
	}
}
