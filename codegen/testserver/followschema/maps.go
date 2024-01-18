package followschema

import (
	"io"
	"strconv"
)

type MapNested struct {
	Value CustomScalar
}

type CustomScalar struct {
	value int64
}

func (s *CustomScalar) UnmarshalGQL(v interface{}) (err error) {
	switch v := v.(type) {
	case string:
		s.value, err = strconv.ParseInt(v, 10, 64)
	case int64:
		s.value = v
	}
	return
}

func (s CustomScalar) MarshalGQL(w io.Writer) {
	_, _ = w.Write([]byte(strconv.Quote(strconv.FormatInt(s.value, 10))))
}
