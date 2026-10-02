package executor_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/validator/rules"

	"github.com/99designs/gqlgen/graphql/executor/testexecutor"
)

// SetDefaultRulesFn lets a server supply its own validation rule set. parseQuery
// calls it once per request and then adjusts the result for the current
// disableSuggestion setting, removing the suggesting variants of two rules and
// adding the silent ones, or the reverse.
//
// Whether that adjustment is observable depends entirely on what the supplied
// function returns. A function returning a fresh set per call leaves the
// "reverse" direction as dead code, because a fresh default set already holds
// the suggesting variants and AddRule is idempotent. A function returning one
// shared set makes it load-bearing: the adjustment made for one request
// persists into the next, and only the reverse direction undoes it.
//
// That second case is what these tests cover. It is the one where a bug leaks
// one request's suggestion setting into another request's error messages.

const (
	suggestingMessage = "input:1:2: Cannot query field \"nam\" on type \"Query\". " +
		"Did you mean \"name\"?\n"
	silentMessage = "input:1:2: Cannot query field \"nam\" on type \"Query\".\n"
)

// A fresh rule set per call is the straightforward configuration, and the
// suggestion setting has to be honoured on each request regardless.
func TestDefaultRulesFnWithAFreshSetPerCall(t *testing.T) {
	t.Parallel()

	calls := 0
	exec := testexecutor.New()
	exec.SetDefaultRulesFn(func() *rules.Rules {
		calls++
		return rules.NewDefaultRules()
	})

	resp := query(exec, "", "{nam}")
	require.Len(t, resp.Errors, 1)
	assert.Equal(t, suggestingMessage, resp.Errors.Error())

	exec.SetDisableSuggestion(true)
	resp = query(exec, "", "{nam}")
	require.Len(t, resp.Errors, 1)
	assert.Equal(t, silentMessage, resp.Errors.Error())

	// The supplied function is consulted per request rather than once and
	// cached, which is what lets a server vary its rules over time.
	assert.Equal(t, 2, calls, "the rules function should be called once per request")
}

// The contract that matters: with one shared rule set, turning suggestions off
// and then on again must restore them. A server that only ever strips the
// suggesting rules would silence every later request, having been asked to
// silence one.
func TestDefaultRulesFnWithASharedSetDoesNotLeakBetweenRequests(t *testing.T) {
	t.Parallel()

	// One instance, handed back on every call, so every adjustment accumulates.
	shared := rules.NewDefaultRules()
	exec := testexecutor.New()
	exec.SetDefaultRulesFn(func() *rules.Rules { return shared })

	// Suggestions on, which is the default.
	resp := query(exec, "", "{nam}")
	require.Len(t, resp.Errors, 1)
	assert.Equal(t, suggestingMessage, resp.Errors.Error())

	// Off: the suggesting rules are swapped out of the shared set.
	exec.SetDisableSuggestion(true)
	resp = query(exec, "", "{nam}")
	require.Len(t, resp.Errors, 1)
	assert.Equal(t, silentMessage, resp.Errors.Error())

	// On again. Nothing rebuilt the set, so this only works if the reverse
	// adjustment puts the suggesting rules back.
	exec.SetDisableSuggestion(false)
	resp = query(exec, "", "{nam}")
	require.Len(t, resp.Errors, 1)
	assert.Equal(t, suggestingMessage, resp.Errors.Error(),
		"re-enabling suggestions must restore them in a shared rule set")

	// And the swap must be repeatable rather than working once in each
	// direction, because a long-lived server does this on every request.
	for range 3 {
		exec.SetDisableSuggestion(true)
		assert.Equal(t, silentMessage, query(exec, "", "{nam}").Errors.Error())
		exec.SetDisableSuggestion(false)
		assert.Equal(t, suggestingMessage, query(exec, "", "{nam}").Errors.Error())
	}
}

// The same leak, checked on the other rule the adjustment touches. ScalarLeafs
// reports selecting an object without a sub-selection, and it has its own
// suggesting and silent variants, so it can regress independently.
func TestDefaultRulesFnSharedSetRestoresScalarLeafsSuggestions(t *testing.T) {
	t.Parallel()

	shared := rules.NewDefaultRules()
	exec := testexecutor.New()
	exec.SetDefaultRulesFn(func() *rules.Rules { return shared })

	// `user` is an object type, so selecting it bare is a ScalarLeafs error.
	const q = "{ user }"

	withSuggestions := query(exec, "", q).Errors.Error()
	require.NotEmpty(t, withSuggestions)

	exec.SetDisableSuggestion(true)
	silent := query(exec, "", q).Errors.Error()
	require.NotEmpty(t, silent)
	require.NotEqual(t, withSuggestions, silent,
		"disabling suggestions should change the ScalarLeafs message")

	exec.SetDisableSuggestion(false)
	assert.Equal(t, withSuggestions, query(exec, "", q).Errors.Error(),
		"re-enabling suggestions must restore the ScalarLeafs message")
}

// A rules function supplying a reduced set proves the executor validates
// against what it was given rather than against its own defaults. The rule
// dropped here is one the suggestion adjustment does not touch.
func TestDefaultRulesFnCanRemoveAnUnrelatedRule(t *testing.T) {
	t.Parallel()

	// An unused fragment, which NoUnusedFragments rejects.
	const q = "{ name } fragment Unused on Query { name }"

	withRule := testexecutor.New()
	require.NotEmpty(t, query(withRule, "", q).Errors,
		"the default rules should reject an unused fragment")

	reduced := rules.NewDefaultRules()
	reduced.RemoveRule("NoUnusedFragments")
	without := testexecutor.New()
	without.SetDefaultRulesFn(func() *rules.Rules { return reduced })

	resp := query(without, "", q)
	assert.Empty(t, resp.Errors,
		"with NoUnusedFragments removed, an unused fragment should be accepted")
}

// The two rules the suggestion adjustment owns are reinstated whatever the
// rules function returns, because the adjustment adds back whichever variant
// matches the current setting. A supplied set cannot opt out of them, and a
// server relying on that would otherwise silently stop rejecting unknown
// fields.
func TestSuggestionAdjustmentReinstatesItsOwnRules(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		disableSuggestion bool
		wantUnknownField  string
	}{
		"with suggestions": {disableSuggestion: false, wantUnknownField: suggestingMessage},
		"without suggestions": {
			disableSuggestion: true,
			wantUnknownField:  silentMessage,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Both rules the adjustment manages are stripped from the supplied
			// set, in both their suggesting and silent spellings.
			stripped := rules.NewDefaultRules()
			for _, rule := range []string{
				"FieldsOnCorrectType", "FieldsOnCorrectTypeWithoutSuggestions",
				"ScalarLeafs", "ScalarLeafsWithoutSuggestions",
			} {
				stripped.RemoveRule(rule)
			}

			exec := testexecutor.New()
			exec.SetDefaultRulesFn(func() *rules.Rules { return stripped })
			exec.SetDisableSuggestion(tc.disableSuggestion)

			unknownField := query(exec, "", "{nam}")
			require.Len(t, unknownField.Errors, 1,
				"FieldsOnCorrectType should have been reinstated")
			assert.Equal(t, tc.wantUnknownField, unknownField.Errors.Error())

			bareObject := query(exec, "", "{ user }")
			assert.NotEmpty(t, bareObject.Errors,
				"ScalarLeafs should have been reinstated")
		})
	}
}
