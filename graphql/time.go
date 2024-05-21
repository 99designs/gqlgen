package graphql

import (
	"errors"
	"io"
	"strconv"
	"time"
)

func MarshalTime(t time.Time) Marshaler {
	if t.IsZero() {
		return Null
	}

	return WriterFunc(func(w io.Writer) {
		io.WriteString(w, strconv.Quote(t.Format(time.RFC3339Nano)))
	})
}

func UnmarshalTime(v interface{}) (time.Time, error) {
	formats := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999",
		"2006-01-02",
	}
	if tmpStr, ok := v.(string); ok {
		for _, f := range formats {
			t, err := time.Parse(f, tmpStr)
			if err == nil {
				return t, nil
			}
		}
	}
	return time.Time{}, errors.New("time is not a string in a recognized format")
}
