package singlefile

import (
	"encoding/json"
	"errors"
	"io"
	"regexp"
)

var re = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

type Email string

func (value *Email) UnmarshalGQL(v any) error {
	input, ok := v.(string)
	if !ok {
		return errors.New("email expects a string value")
	}
	if !re.MatchString(input) {
		return errors.New("invalid email format")
	}
	*value = Email(input)
	return nil
}

func (value Email) MarshalGQL(w io.Writer) {
	output, _ := json.Marshal(string(value))
	w.Write(output)
}
