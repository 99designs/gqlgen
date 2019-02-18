package starwars

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type CharacterFields struct {
	ID        string
	Name      string
	FriendIds []string
	AppearsIn []Episode
}

type Human struct {
	CharacterFields
	StarshipIds  []string
	heightMeters float64
	Mass         float64
}

func (h *Human) Height(unit LengthUnit) float64 {
	switch unit {
	case "METER", "":
		return h.heightMeters
	case "FOOT":
		return h.heightMeters * 3.28084
	default:
		panic("invalid unit")
	}
}

func (Human) IsCharacter()    {}
func (Human) IsSearchResult() {}

type Review struct {
	Stars      int
	Commentary *string
	Time       time.Time
}

type Droid struct {
	CharacterFields
	PrimaryFunction string
}

func (Droid) IsCharacter()    {}
func (Droid) IsSearchResult() {}

func (r *Resolver) resolveFriendConnection(ctx context.Context, ids []string, first *int, after *string) (*FriendsConnection, error) {
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

	return &FriendsConnection{
		ids:  ids,
		from: from,
		to:   to,
	}, nil
}

type FriendsConnection struct {
	ids  []string
	from int
	to   int
}

func (f *FriendsConnection) TotalCount() int {
	return len(f.ids)
}

func (f *FriendsConnection) PageInfo() PageInfo {
	return PageInfo{
		StartCursor: encodeCursor(f.from),
		EndCursor:   encodeCursor(f.to - 1),
		HasNextPage: f.to < len(f.ids),
	}
}

func encodeCursor(i int) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("cursor%d", i+1)))
}
