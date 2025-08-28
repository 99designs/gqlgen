package graphql

import (
	"errors"
	"time"

	dur "github.com/sosodev/duration"
)

// UnmarshalDuration returns the duration from a string in ISO8601 format
// PnDTnHnMn.nS with days considered to be exactly 24 hours.
// See https://en.wikipedia.org/wiki/ISO_8601#Durations
// P - Period
// D - D is the
// T - T is the time designator that precedes the time components
// H - H is the hour designator that follows the value for the number of hours.
// M - M is the minute designator that follows the value for the number of minutes.
// S - S is the second designator that follows the value for the number of seconds.
// "PT20.345S" -- parses as "20.345 seconds"
// "PT15M"     -- parses as "15 minutes" (where a minute is 60 seconds)
// "PT10H"     -- parses as "10 hours" (where an hour is 3600 seconds)
// "P2D"       -- parses as "2 days" (where a day is 24 hours or 86400 seconds)
func UnmarshalDuration(v any) (time.Duration, error) {
	input, ok := v.(string)
	if !ok {
		return 0, errors.New("input must be a string")
	}

	d2, err := dur.Parse(input)
	if err != nil {
		return 0, err
	}
	return d2.ToTimeDuration(), nil
}

// MarshalDuration returns the duration in ISO8601 format
// PnDTnHnMn.nS with days considered to be exactly 24 hours.
// See https://en.wikipedia.org/wiki/ISO_8601#Durations
// P - Period
// D - D is the
// T - T is the time designator that precedes the time components
// H - H is the hour designator that follows the value for the number of hours.
// M - M is the minute designator that follows the value for the number of minutes.
// S - S is the second designator that follows the value for the number of seconds.
// "PT20.345S" -- parses as "20.345 seconds"
// "PT15M"     -- parses as "15 minutes" (where a minute is 60 seconds)
// "PT10H"     -- parses as "10 hours" (where an hour is 3600 seconds)
// "P2D"       -- parses as "2 days" (where a day is 24 hours or 86400 seconds)
func MarshalDuration(d time.Duration) Marshaler {
	return MarshalString(dur.Format(d))
}
