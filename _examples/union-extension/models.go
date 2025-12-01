package unionextension

import (
	"io"

	"github.com/99designs/gqlgen/graphql"
)

type CachedLike struct{}

type CachedPost struct{}

func (CachedLike) IsEvent() {}
func (CachedPost) IsEvent() {}

var (
	_ graphql.Marshaler = CachedLike{}
	_ graphql.Marshaler = CachedPost{}
)

// This needs to be used with caution, as the returned data doesn't go through the usual field resolution process.
// This will be returned for all users, no matter what fields are requested.
func (CachedLike) MarshalGQL(w io.Writer) {
	w.Write([]byte(`{"from":"CachedLike"}`))
}

// This needs to be used with caution, as the returned data doesn't go through the usual field resolution process.
// This will be returned for all users, no matter what fields are requested.
func (CachedPost) MarshalGQL(w io.Writer) {
	w.Write([]byte(`{"message":"CachedPost"}`))
}
