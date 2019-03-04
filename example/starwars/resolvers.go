//go:generate rm -rf generated
//go:generate go run ../../testdata/gqlgen.go

package starwars

import (
	"context"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/99designs/gqlgen/example/starwars/generated"

	"github.com/99designs/gqlgen/example/starwars/models"
)

type Resolver struct {
	humans    map[string]models.Human
	droid     map[string]models.Droid
	starships map[string]models.Starship
	reviews   map[models.Episode][]models.Review
}

func (r *Resolver) Droid() generated.DroidResolver {
	return &droidResolver{r}
}

func (r *Resolver) FriendsConnection() generated.FriendsConnectionResolver {
	return &friendsConnectionResolver{r}
}

func (r *Resolver) Human() generated.HumanResolver {
	return &humanResolver{r}
}

func (r *Resolver) Mutation() generated.MutationResolver {
	return &mutationResolver{r}
}

func (r *Resolver) Query() generated.QueryResolver {
	return &queryResolver{r}
}

func (r *Resolver) Starship() generated.StarshipResolver {
	return &starshipResolver{r}
}

func (r *Resolver) resolveCharacters(ctx context.Context, ids []string) ([]models.Character, error) {
	var result []models.Character
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

func (r *droidResolver) Friends(ctx context.Context, obj *models.Droid) ([]models.Character, error) {
	return r.resolveCharacters(ctx, obj.FriendIds)
}

func (r *droidResolver) FriendsConnection(ctx context.Context, obj *models.Droid, first *int, after *string) (*models.FriendsConnection, error) {
	return r.resolveFriendConnection(ctx, obj.FriendIds, first, after)
}

type friendsConnectionResolver struct{ *Resolver }

func (r *Resolver) resolveFriendConnection(ctx context.Context, ids []string, first *int, after *string) (*models.FriendsConnection, error) {
	from := 0
	if after != nil {
		b, err := base64.StdEncoding.DecodeString(*after)
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
	if first != nil {
		to = from + *first
		if to > len(ids) {
			to = len(ids)
		}
	}

	return &models.FriendsConnection{
		Ids:  ids,
		From: from,
		To:   to,
	}, nil
}

func (r *friendsConnectionResolver) Edges(ctx context.Context, obj *models.FriendsConnection) ([]models.FriendsEdge, error) {
	friends, err := r.resolveCharacters(ctx, obj.Ids)
	if err != nil {
		return nil, err
	}

	edges := make([]models.FriendsEdge, obj.To-obj.From)
	for i := range edges {
		edges[i] = models.FriendsEdge{
			Cursor: models.EncodeCursor(obj.From + i),
			Node:   friends[obj.From+i],
		}
	}
	return edges, nil
}

func (r *friendsConnectionResolver) Friends(ctx context.Context, obj *models.FriendsConnection) ([]models.Character, error) {
	return r.resolveCharacters(ctx, obj.Ids)
}

type humanResolver struct{ *Resolver }

func (r *humanResolver) Friends(ctx context.Context, obj *models.Human) ([]models.Character, error) {
	return r.resolveCharacters(ctx, obj.FriendIds)
}

func (r *humanResolver) FriendsConnection(ctx context.Context, obj *models.Human, first *int, after *string) (*models.FriendsConnection, error) {
	return r.resolveFriendConnection(ctx, obj.FriendIds, first, after)
}

func (r *humanResolver) Starships(ctx context.Context, obj *models.Human) ([]models.Starship, error) {
	var result []models.Starship
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

func (r *mutationResolver) CreateReview(ctx context.Context, episode models.Episode, review models.Review) (*models.Review, error) {
	review.Time = time.Now()
	time.Sleep(1 * time.Second)
	r.reviews[episode] = append(r.reviews[episode], review)
	return &review, nil
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Hero(ctx context.Context, episode *models.Episode) (models.Character, error) {
	if *episode == models.EpisodeEmpire {
		return r.humans["1000"], nil
	}
	return r.droid["2001"], nil
}

func (r *queryResolver) Reviews(ctx context.Context, episode models.Episode, since *time.Time) ([]models.Review, error) {
	if since == nil {
		return r.reviews[episode], nil
	}

	var filtered []models.Review
	for _, rev := range r.reviews[episode] {
		if rev.Time.After(*since) {
			filtered = append(filtered, rev)
		}
	}
	return filtered, nil
}

func (r *queryResolver) Search(ctx context.Context, text string) ([]models.SearchResult, error) {
	var l []models.SearchResult
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

func (r *queryResolver) Character(ctx context.Context, id string) (models.Character, error) {
	if h, ok := r.humans[id]; ok {
		return &h, nil
	}
	if d, ok := r.droid[id]; ok {
		return &d, nil
	}
	return nil, nil
}

func (r *queryResolver) Droid(ctx context.Context, id string) (*models.Droid, error) {
	if d, ok := r.droid[id]; ok {
		return &d, nil
	}
	return nil, nil
}

func (r *queryResolver) Human(ctx context.Context, id string) (*models.Human, error) {
	if h, ok := r.humans[id]; ok {
		return &h, nil
	}
	return nil, nil
}

func (r *queryResolver) Starship(ctx context.Context, id string) (*models.Starship, error) {
	if s, ok := r.starships[id]; ok {
		return &s, nil
	}
	return nil, nil
}

type starshipResolver struct{ *Resolver }

func (r *starshipResolver) Length(ctx context.Context, obj *models.Starship, unit *models.LengthUnit) (float64, error) {
	switch *unit {
	case models.LengthUnitMeter, "":
		return obj.Length, nil
	case models.LengthUnitFoot:
		return obj.Length * 3.28084, nil
	default:
		return 0, errors.New("invalid unit")
	}
}

func NewResolver() generated.Config {
	r := Resolver{}
	r.humans = map[string]models.Human{
		"1000": {
			CharacterFields: models.CharacterFields{
				ID:        "1000",
				Name:      "Luke Skywalker",
				FriendIds: []string{"1002", "1003", "2000", "2001"},
				AppearsIn: []models.Episode{models.EpisodeNewhope, models.EpisodeEmpire, models.EpisodeJedi},
			},
			HeightMeters: 1.72,
			Mass:         77,
			StarshipIds:  []string{"3001", "3003"},
		},
		"1001": {
			CharacterFields: models.CharacterFields{
				ID:        "1001",
				Name:      "Darth Vader",
				FriendIds: []string{"1004"},
				AppearsIn: []models.Episode{models.EpisodeNewhope, models.EpisodeEmpire, models.EpisodeJedi},
			},
			HeightMeters: 2.02,
			Mass:         136,
			StarshipIds:  []string{"3002"},
		},
		"1002": {
			CharacterFields: models.CharacterFields{
				ID:        "1002",
				Name:      "Han Solo",
				FriendIds: []string{"1000", "1003", "2001"},
				AppearsIn: []models.Episode{models.EpisodeNewhope, models.EpisodeEmpire, models.EpisodeJedi},
			},
			HeightMeters: 1.8,
			Mass:         80,
			StarshipIds:  []string{"3000", "3003"},
		},
		"1003": {
			CharacterFields: models.CharacterFields{
				ID:        "1003",
				Name:      "Leia Organa",
				FriendIds: []string{"1000", "1002", "2000", "2001"},
				AppearsIn: []models.Episode{models.EpisodeNewhope, models.EpisodeEmpire, models.EpisodeJedi},
			},
			HeightMeters: 1.5,
			Mass:         49,
		},
		"1004": {
			CharacterFields: models.CharacterFields{
				ID:        "1004",
				Name:      "Wilhuff Tarkin",
				FriendIds: []string{"1001"},
				AppearsIn: []models.Episode{models.EpisodeNewhope},
			},
			HeightMeters: 1.8,
			Mass:         0,
		},
	}

	r.droid = map[string]models.Droid{
		"2000": {
			CharacterFields: models.CharacterFields{
				ID:        "2000",
				Name:      "C-3PO",
				FriendIds: []string{"1000", "1002", "1003", "2001"},
				AppearsIn: []models.Episode{models.EpisodeNewhope, models.EpisodeEmpire, models.EpisodeJedi},
			},
			PrimaryFunction: "Protocol",
		},
		"2001": {
			CharacterFields: models.CharacterFields{
				ID:        "2001",
				Name:      "R2-D2",
				FriendIds: []string{"1000", "1002", "1003"},
				AppearsIn: []models.Episode{models.EpisodeNewhope, models.EpisodeEmpire, models.EpisodeJedi},
			},
			PrimaryFunction: "Astromech",
		},
	}

	r.starships = map[string]models.Starship{
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

	r.reviews = map[models.Episode][]models.Review{}

	return generated.Config{
		Resolvers: &r,
	}
}
