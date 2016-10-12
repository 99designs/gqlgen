package graphql

import "testing"

type testStringResolver struct{}

func (r *testStringResolver) Hello() string {
	return "Hello world!"
}

func TestString(t *testing.T) {
	schema, err := NewSchema(`
		type Query {
    	hello: String
  	}
	`, "test", &testStringResolver{})
	if err != nil {
		t.Fatal(err)
	}

	got, err := schema.Exec(`{ hello }`)
	if err != nil {
		t.Fatal(err)
	}

	if want := "Hello world!"; got != want {
		t.Errorf("want %#v, got %#v", want, got)
	}
}
