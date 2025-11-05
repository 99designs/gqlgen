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

func UnmarshalTime(v any) (time.Time, error) {
	if v == nil {
		return time.Time{}, nil
	}

	if tmpStr, ok := v.(string); ok {
		if tmpStr == "" {
			return time.Time{}, nil
		}

		t, err := time.Parse(time.RFC3339Nano, tmpStr)
		if err == nil {
			return t, nil
		}
		t, err = time.Parse(time.RFC3339, tmpStr)
		if err == nil {
			return t, nil
		}
		t, err = time.Parse(time.DateTime, tmpStr)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, errors.New("time should be RFC3339Nano formatted string")
}

func MarshalDate(t time.Time) Marshaler {
	if t.IsZero() {
		return Null
	}

	return WriterFunc(func(w io.Writer) {
		io.WriteString(w, strconv.Quote(t.Format(time.DateOnly)))
	})
}

func UnmarshalDate(v any) (time.Time, error) {
	if v == nil {
		return time.Time{}, nil
	}

	if tmpStr, ok := v.(string); ok {
		if tmpStr == "" {
			return time.Time{}, nil
		}

		t, err := time.Parse(time.DateOnly, tmpStr)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, errors.New("time should be DateOnly formatted string")
}
