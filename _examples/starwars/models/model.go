package models

import (
	"encoding/base64"
	"fmt"
	"time"
)

type CharacterFields struct {
	ID        string
	Name      string
	FriendIds []string
	AppearsIn []Episode
}

func (cf CharacterFields) GetID() string                            { return cf.ID }
func (cf CharacterFields) GetName() string                          { return cf.Name }
func (cf CharacterFields) GetAppearsIn() []Episode                  { return cf.AppearsIn }
func (cf CharacterFields) GetFriendsConnection() *FriendsConnection { return nil } // Handled in resolver
func (cf CharacterFields) GetFriends() []Character                  { return nil } // Handled in resolver

type Human struct {
	CharacterFields
	StarshipIds  []string
	HeightMeters float64
	Mass         float64
}

func (h *Human) Height(unit LengthUnit) float64 {
	switch unit {
	case "METER", "":
		return h.HeightMeters
	case "FOOT":
		return h.HeightMeters * 3.28084
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

type FriendsConnection struct {
	Ids  []string
	From int
	To   int
}

func (f *FriendsConnection) TotalCount() int {
	return len(f.Ids)
}

func (f *FriendsConnection) PageInfo() PageInfo {
	return PageInfo{
		StartCursor: EncodeCursor(f.From),
		EndCursor:   EncodeCursor(f.To - 1),
		HasNextPage: f.To < len(f.Ids),
	}
}

func EncodeCursor(i int) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("cursor%d", i+1)))
}
