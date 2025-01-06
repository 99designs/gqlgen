package model

import (
	"encoding/json"
	"fmt"
)

// this file is provided as an example for int-based enums
// but if you instead wanted to support arbitrary 
// english words for numbers to integers, consider
// github.com/will-lol/numberconverter/etoi or a similar library
type IntTyped int

const (
	IntTypedOne IntTyped = iota + 1
	IntTypedTwo
)

const (
	IntUntypedOne = iota + 1
	IntUntypedTwo
)

func (t IntTyped) String() string {
	switch t {
	case IntTypedOne:
		return "ONE"
	case IntTypedTwo:
		return "TWO"
	default:
		return "UNKNOWN"
	}
}

func (t IntTyped) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, t.String())), nil
}

func (t *IntTyped) UnmarshalJSON(b []byte) (err error) {
	var s string

	if err = json.Unmarshal(b, &s); err != nil {
		return err
	}

	switch s {
	case "ONE":
		*t = IntTypedOne
	case "TWO":
		*t = IntTypedTwo
	default:
		return fmt.Errorf("unexpected enum value %q", s)
	}

	return nil
}

type StringTyped string

const (
	StringTypedOne StringTyped = "ONE"
	StringTypedTwo StringTyped = "TWO"
)

const (
	StringUntypedOne = "ONE"
	StringUntypedTwo = "TWO"
)

type BoolTyped bool

const (
	BoolTypedTrue  BoolTyped = true
	BoolTypedFalse BoolTyped = false
)

const (
	BoolUntypedTrue  = true
	BoolUntypedFalse = false
)

type VarTyped bool

var (
	VarTypedTrue  VarTyped = true
	VarTypedFalse VarTyped = false
)

var (
	VarUntypedTrue  = true
	VarUntypedFalse = false
)
