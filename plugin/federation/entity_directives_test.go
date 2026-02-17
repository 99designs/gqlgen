package federation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen"
)

func TestEntity_HasObjectDirectives(t *testing.T) {
	assert.False(t, (&Entity{}).HasObjectDirectives())
	assert.False(t, (&Entity{ImplDirectives: nil}).HasObjectDirectives())
	assert.True(
		t,
		(&Entity{ImplDirectives: []*codegen.Directive{{Name: "guard"}}}).HasObjectDirectives(),
	)
}

func TestEntity_NonFederationDirectives(t *testing.T) {
	tests := []struct {
		name      string
		entity    *Entity
		wantNames []string
	}{
		{
			name:      "nil Def",
			entity:    &Entity{Def: nil},
			wantNames: nil,
		},
		{
			name: "filters all federation directives",
			entity: &Entity{Def: &ast.Definition{
				Directives: ast.DirectiveList{
					{Name: "key"},
					{Name: "requires"},
					{Name: "provides"},
					{Name: "external"},
					{Name: "extends"},
					{Name: "tag"},
					{Name: "inaccessible"},
					{Name: "shareable"},
					{Name: "interfaceObject"},
					{Name: "override"},
					{Name: "composeDirective"},
					{Name: "link"},
					{Name: "authenticated"},
					{Name: "requiresScopes"},
					{Name: "policy"},
					{Name: "entityResolver"},
					{Name: "goModel"},
					{Name: "goField"},
					{Name: "goTag"},
				},
			}},
			wantNames: nil,
		},
		{
			name: "keeps user directives in order",
			entity: &Entity{Def: &ast.Definition{
				Directives: ast.DirectiveList{
					{Name: "key"},
					{Name: "guard"},
					{Name: "shareable"},
					{Name: "hasRole"},
				},
			}},
			wantNames: []string{"guard", "hasRole"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.entity.NonFederationDirectives()
			if tt.wantNames == nil {
				assert.Nil(t, got)
				return
			}
			require.Len(t, got, len(tt.wantNames))
			for i, d := range got {
				assert.Equal(t, tt.wantNames[i], d.Name)
			}
		})
	}
}

func TestFederationDirectiveNames_Completeness(t *testing.T) {
	for _, name := range []string{
		"key", "requires", "provides", "external", "extends",
		"tag", "inaccessible", "shareable", "interfaceObject",
		"override", "composeDirective", "link",
		"authenticated", "requiresScopes", "policy",
		"entityResolver", "goModel", "goField", "goTag",
	} {
		assert.True(t, federationDirectiveNames[name], "%q missing", name)
	}
}

// TestEntity_NonFederationDirectives_Filtering validates that user-defined
// directives are preserved while federation directives are filtered.
func TestEntity_NonFederationDirectives_Filtering(t *testing.T) {
	tests := []struct {
		name      string
		entity    *Entity
		wantNames []string
		desc      string
	}{
		{
			name: "user directive preserved",
			entity: &Entity{Def: &ast.Definition{
				Directives: ast.DirectiveList{
					{Name: "key"},
					{Name: "guard"},
				},
			}},
			wantNames: []string{"guard"},
			desc:      "User-defined @guard directive should be preserved while @key is filtered",
		},
		{
			name: "multiple user directives",
			entity: &Entity{Def: &ast.Definition{
				Directives: ast.DirectiveList{
					{Name: "shareable"},
					{Name: "auth"},
					{Name: "guard"},
					{Name: "key"},
				},
			}},
			wantNames: []string{"auth", "guard"},
			desc:      "Multiple user directives should be preserved in order",
		},
		{
			name: "only federation directives",
			entity: &Entity{Def: &ast.Definition{
				Directives: ast.DirectiveList{
					{Name: "key"},
					{Name: "shareable"},
					{Name: "tag"},
				},
			}},
			wantNames: nil,
			desc:      "Only federation directives should result in empty list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.entity.NonFederationDirectives()
			if tt.wantNames == nil {
				assert.Nil(t, got, tt.desc)
				return
			}
			require.Len(t, got, len(tt.wantNames), tt.desc)
			for i, d := range got {
				assert.Equal(t, tt.wantNames[i], d.Name)
			}
		})
	}
}
