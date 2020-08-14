package cache

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/stretchr/testify/require"

	"github.com/vektah/gqlparser/v2/ast"
)

func TestCacheControl_SetCacheHint(t *testing.T) {
	t.Parallel()

	createFieldContext := func(alias string) *graphql.FieldContext {
		return &graphql.FieldContext{
			Field: graphql.CollectedField{
				Field: &ast.Field{
					Alias: alias,
				},
			},
		}
	}

	t.Run("should add hint in context", func(t *testing.T) {
		fooCtx := createFieldContext("foo")

		ctx := graphql.WithFieldContext(graphql.WithResponseContext(context.Background(), nil, nil), fooCtx)
		ctx = WithCacheControlExtension(ctx)

		SetHint(ctx, ScopePublic, time.Minute)

		c := CacheControl(ctx)
		require.Equal(t, 1, c.Version)
		require.Equal(t, fooCtx.Path(), c.Hints[0].Path)
		require.Equal(t, time.Minute.Seconds(), c.Hints[0].MaxAge)
		require.Equal(t, ScopePublic, c.Hints[0].Scope)
	})

	t.Run("should not add hint in context", func(t *testing.T) {
		fooCtx := createFieldContext("foo")

		ctx := graphql.WithFieldContext(graphql.WithResponseContext(context.Background(), nil, nil), fooCtx)

		SetHint(ctx, ScopePublic, time.Minute)

		c := CacheControl(ctx)
		require.Nil(t, c)
	})

}

func TestCacheControl_OverallPolicy(t *testing.T) {
	type fields struct {
		Version int
		Hints   []Hint
	}
	tests := []struct {
		name   string
		fields fields
		want   OverallCachePolicy
	}{
		{
			name: "one hint public",
			fields: fields{
				Version: 1,
				Hints: []Hint{{
					Path:   nil,
					MaxAge: (10 * time.Second).Seconds(),
					Scope:  ScopePublic,
				}},
			},
			want: OverallCachePolicy{
				MaxAge: (10 * time.Second).Seconds(),
				Scope:  ScopePublic,
			},
		},
		{
			name: "one hint private",
			fields: fields{
				Version: 1,
				Hints: []Hint{{
					Path:   nil,
					MaxAge: (5 * time.Second).Seconds(),
					Scope:  ScopePrivate,
				}},
			},
			want: OverallCachePolicy{
				MaxAge: (5 * time.Second).Seconds(),
				Scope:  ScopePrivate,
			},
		},
		{
			name: "multiple hints with one of them is private",
			fields: fields{
				Version: 1,
				Hints: []Hint{
					{
						Path:   nil,
						MaxAge: (5 * time.Second).Seconds(),
						Scope:  ScopePublic,
					},
					{
						Path:   nil,
						MaxAge: (10 * time.Second).Seconds(),
						Scope:  ScopePrivate,
					},
				},
			},
			want: OverallCachePolicy{
				MaxAge: (5 * time.Second).Seconds(),
				Scope:  ScopePrivate,
			},
		},
		{
			name: "multiple hints all of them are public",
			fields: fields{
				Version: 1,
				Hints: []Hint{
					{
						Path:   nil,
						MaxAge: (5 * time.Second).Seconds(),
						Scope:  ScopePublic,
					},
					{
						Path:   nil,
						MaxAge: (1 * time.Second).Seconds(),
						Scope:  ScopePublic,
					},
					{
						Path:   nil,
						MaxAge: (10 * time.Second).Seconds(),
						Scope:  ScopePublic,
					},
				},
			},
			want: OverallCachePolicy{
				MaxAge: (1 * time.Second).Seconds(),
				Scope:  ScopePublic,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := CacheControlExtension{
				Version: tt.fields.Version,
				Hints:   tt.fields.Hints,
			}
			if got := cache.OverallPolicy(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OverallPolicy() = %v, want %v", got, tt.want)
			}
		})
	}
}
