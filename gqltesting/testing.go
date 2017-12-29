package gqltesting

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"testing"

	graphql "github.com/neelance/graphql-go"
)

// Test is a GraphQL test case to be used with RunTest(s).
type Test struct {
	Context        context.Context
	Schema         *graphql.Schema
	Query          string
	OperationName  string
	Variables      map[string]interface{}
	ExpectedResult string
}

// RunTests runs the given GraphQL test cases as subtests.
func RunTests(t *testing.T, tests []*Test) {
	if len(tests) == 1 {
		RunTest(t, tests[0])
		return
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			RunTest(t, test)
		})
	}
}

// RunTest runs a single GraphQL test case.
func RunTest(t *testing.T, test *Test) {
	if test.Context == nil {
		test.Context = context.Background()
	}
	result := test.Schema.Exec(test.Context, test.Query, test.OperationName, test.Variables)
	if len(result.Errors) != 0 {
		t.Fatal(result.Errors[0])
	}
	got := formatJSON(t, result.Data)

	want := formatJSON(t, []byte(test.ExpectedResult))

	if !bytes.Equal(got, want) {
		t.Logf("got:  %s", got)
		t.Logf("want: %s", want)
		t.Fail()
	}
}

func formatJSON(t *testing.T, data []byte) []byte {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("invalid JSON: %s", err)
	}
	formatted, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return formatted
}
