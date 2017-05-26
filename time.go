package graphql

import (
	"fmt"
	"time"
)

// Time is a custom GraphQL type to represent an instant in time. It has to be added to a schema
// via "scalar Time" since it is not a predeclared GraphQL type like "ID".
type Time struct {
	time.Time
}

func (_ Time) ImplementsGraphQLType(name string) bool {
	return name == "Time"
}

func (t *Time) UnmarshalGraphQL(input interface{}) error {
	switch input := input.(type) {
	case time.Time:
		t.Time = input
		return nil
	case string:
		var err error
		t.Time, err = time.Parse(time.RFC3339, input)
		return err
	case int:
		t.Time = time.Unix(int64(input), 0)
		return nil
	case float64:
		t.Time = time.Unix(int64(input), 0)
		return nil
	default:
		return fmt.Errorf("wrong type")
	}
}
