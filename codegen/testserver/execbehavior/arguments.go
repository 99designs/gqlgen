package execbehavior

import (
	"errors"
	"io"
	"strconv"
)

// Strict is a scalar that rejects the string "bad".
type Strict string

func (s *Strict) UnmarshalGQL(v any) error {
	str, ok := v.(string)
	if !ok || str == "bad" {
		return errors.New("strict rejects this value")
	}
	*s = Strict(str)
	return nil
}

func (s Strict) MarshalGQL(w io.Writer) {
	_, _ = io.WriteString(w, strconv.Quote(string(s)))
}
