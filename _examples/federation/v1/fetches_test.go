package main_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	accounts "github.com/99designs/gqlgen/_examples/federation/v1/accounts/schema"
	content "github.com/99designs/gqlgen/_examples/federation/v1/content/schema"
	progress "github.com/99designs/gqlgen/_examples/federation/v1/progress/schema"
	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

func newClient(schema graphql.ExecutableSchema) *client.Client {
	srv := handler.New(schema)
	srv.AddTransport(transport.POST{})
	srv.Use(extension.Introspection{})
	return client.New(srv)
}

// A Federation v1 gateway built on @apollo/query-planner sends these fetches to
// the subgraphs for the operations in ../src/__test__/federation-v1.spec.ts. Each
// query is verbatim from @apollo/query-planner 2.12.2, which minifies them.
func TestQueryPlannerFetches(t *testing.T) {
	cases := map[string]struct {
		schema    graphql.ExecutableSchema
		query     string
		variables map[string]any
		want      string
	}{
		"root field": {
			schema: accounts.Schema,
			query:  `query MyProgress__accounts__0{me{__typename id username}}`,
			want:   `{"me": {"__typename": "User", "id": "1", "username": "sal"}}`,
		},
		"entity with a @requires field": {
			schema: progress.Schema,
			query: `query MyProgress__progress__1($representations:[_Any!]!){` +
				`_entities(representations:$representations){...on User{greeting ` +
				`progress(since:"2024-01-01"){percent completedOn video{__typename id title}}}}}`,
			variables: map[string]any{"representations": []any{
				map[string]any{"__typename": "User", "id": "1", "username": "sal"},
			}},
			want: `{"_entities": [{
				"greeting": "Hi, sal!",
				"progress": [{
					"percent": 1,
					"completedOn": "2024-03-04",
					"video": {"__typename": "Video", "id": "video-1", "title": "Adding fractions"}
				}]
			}]}`,
		},
		"entities from a multi resolver": {
			schema: content.Schema,
			query: `query MyProgress__content__2($representations:[_Any!]!){` +
				`_entities(representations:$representations){...on Video{durationSeconds}}}`,
			variables: map[string]any{"representations": []any{
				map[string]any{"__typename": "Video", "id": "video-1"},
				map[string]any{"__typename": "Video", "id": "video-1"},
			}},
			want: `{"_entities": [{"durationSeconds": 300}, {"durationSeconds": 300}]}`,
		},
		"deprecated argument": {
			schema: accounts.Schema,
			query:  `query DeprecatedArgument__accounts__0{me{avatarURL(px:64)}}`,
			want:   `{"me": {"avatarURL": "https://avatars.example.com/1?size=64"}}`,
		},
		"interface, union, fragments, @include and a deprecated argument": {
			schema: content.Schema,
			query: `query Search__content__0($query:String!$withArticles:Boolean!){` +
				`search(query:$query kinds:[VIDEO ARTICLE]limit:5){__typename ` +
				`...on Video{id title}...on Article@include(if:$withArticles){wordCount}}` +
				`content(id:"video-1"){__typename id title ...on Video{publishedOn}}}`,
			variables: map[string]any{"query": "fractions", "withArticles": true},
			want: `{
				"search": [
					{"__typename": "Video", "id": "video-1", "title": "Adding fractions"},
					{"__typename": "Article", "wordCount": 1200}
				],
				"content": {
					"__typename": "Video",
					"id": "video-1",
					"title": "Adding fractions",
					"publishedOn": "2024-01-02"
				}
			}`,
		},
		"mutation with a deprecated argument and input field": {
			schema: accounts.Schema,
			query: `mutation CreateStudents__accounts__0($input:CreateStudentsInput!){` +
				`createStudents(input:$input notify:true){__typename id username}}`,
			variables: map[string]any{"input": map[string]any{
				"usernames": []any{"alex"},
				"listKey":   "period-1",
			}},
			want: `{"createStudents": [
				{"__typename": "User", "id": "period-1-alex", "username": "alex"}
			]}`,
		},
		"entity without @requires fields": {
			schema: progress.Schema,
			query: `query CreateStudents__progress__1($representations:[_Any!]!){` +
				`_entities(representations:$representations){...on User{progress{percent}}}}`,
			variables: map[string]any{"representations": []any{
				map[string]any{"__typename": "User", "id": "period-1-alex"},
			}},
			want: `{"_entities": [{"progress": [{"percent": 1}]}]}`,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var options []client.Option
			for name, value := range tc.variables {
				options = append(options, client.Var(name, value))
			}

			resp, err := newClient(tc.schema).RawPost(tc.query, options...)
			require.NoError(t, err)
			require.Empty(t, resp.Errors)
			data, err := json.Marshal(resp.Data)
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(data))
		})
	}
}

// Composition reads each subgraph's SDL from _service, so it must match the
// schema file exactly: no gqlgen or gqlparser built-ins.
func TestServiceSDL(t *testing.T) {
	cases := map[string]graphql.ExecutableSchema{
		"accounts": accounts.Schema,
		"content":  content.Schema,
		"progress": progress.Schema,
	}
	for name, schema := range cases {
		t.Run(name, func(t *testing.T) {
			want, err := os.ReadFile(filepath.Join(name, "graph", "schema.graphqls"))
			require.NoError(t, err)

			var resp struct {
				Service struct{ SDL string } `json:"_service"`
			}
			newClient(schema).MustPost(`{ _service { sdl } }`, &resp)
			require.Equal(t, string(want), resp.Service.SDL)
		})
	}
}
