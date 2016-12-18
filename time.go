package graphql

import (
	"fmt"
	"time"
)

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
	default:
		return fmt.Errorf("wrong type")
	}
}
