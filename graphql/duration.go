package graphql

import (
	"fmt"
	"time"

	dur "github.com/sosodev/duration"
)

// UnmarshalDuration returns the duration from a string in ISO8601 format
func UnmarshalDuration(v interface{}) (time.Duration, error) {
	input, ok := v.(string)
	if !ok {
		return 0, fmt.Errorf("input must be a string")
	}

	d2, err := dur.Parse(input)
	if err != nil {
		return 0, err
	}
	return d2.ToTimeDuration(), nil
}

// MarshalDuration returns the duration on ISO8601 format
func MarshalDuration(d time.Duration) Marshaler {
	return MarshalString(dur.Format(d))
}
