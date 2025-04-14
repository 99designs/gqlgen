package code

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsAliasInternal(t *testing.T) {
	isInternal := []struct {
		name string
		lhs  string
		rhs  string
	}{
		{
			name: "same root internal alias",
			lhs:  "github.com/org/repo/graph/model",
			rhs:  "github.com/org/repo/internal/store/example",
		},
		{
			name: "same root internal alias, path begins with internal",
			lhs:  "graph/model",
			rhs:  "internal/store/example",
		},
	}

	for _, tc := range isInternal {
		t.Run(tc.name, func(t *testing.T) {
			require.True(t, isAliasInternal(tc.lhs, tc.rhs))
		})
	}

	isNotInternal := []struct {
		name string
		lhs  string
		rhs  string
	}{
		{
			name: "same root not internal alias",
			lhs:  "github.com/org/repo/graph/model",
			rhs:  "github.com/org/repo/store/example",
		},
		{
			name: "same root both internal",
			lhs:  "github.com/org/repo/internal/model",
			rhs:  "github.com/org/repo/internal/store/example",
		},
		{
			name: "diff root not internal alias",
			lhs:  "github.com/org/repo/graph/model",
			rhs:  "github.com/org/repoB/store/example",
		},
		{
			name: "diff root internal alias",
			lhs:  "github.com/org/repo/graph/model",
			rhs:  "github.com/org/repoB/internal/store/example",
		},
	}

	for _, tc := range isNotInternal {
		t.Run(tc.name, func(t *testing.T) {
			require.False(t, isAliasInternal(tc.lhs, tc.rhs))
		})
	}
}
