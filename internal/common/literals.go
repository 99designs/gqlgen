package common

import (
	"strconv"
	"text/scanner"
)

type Literal interface {
	Value() interface{}
}

type BasicLit struct {
	Type rune
	Text string
}

func (lit *BasicLit) Value() interface{} {
	switch lit.Type {
	case scanner.Int, scanner.Float:
		value, err := strconv.ParseFloat(lit.Text, 64)
		if err != nil {
			panic(err)
		}
		return value

	case scanner.String:
		value, err := strconv.Unquote(lit.Text)
		if err != nil {
			panic(err)
		}
		return value

	case scanner.Ident:
		switch lit.Text {
		case "true":
			return true
		case "false":
			return false
		default:
			return lit.Text
		}

	default:
		panic("invalid literal")
	}
}
