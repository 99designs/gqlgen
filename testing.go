package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"testing"
)

type Test struct {
	Schema         *Schema
	Query          string
	OperationName  string
	Variables      map[string]interface{}
	ExpectedResult string
}

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

func RunTest(t *testing.T, test *Test) {
	result := test.Schema.Exec(context.Background(), test.Query, test.OperationName, test.Variables)
	if len(result.Errors) != 0 {
		t.Fatal(result.Errors[0])
	}

	got, err := json.Marshal(result.Data)
	if err != nil {
		t.Fatal(err)
	}

	var v interface{}
	if err := json.Unmarshal([]byte(test.ExpectedResult), &v); err != nil {
		t.Fatalf("invalid JSON for ExpectedResult: %s", err)
	}
	want, _ := json.Marshal(v)

	if !bytes.Equal(got, want) {
		t.Logf("want: %s", want)
		t.Logf("got:  %s", got)
		t.Fail()
	}
}
