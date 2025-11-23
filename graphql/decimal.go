package graphql

import (
	"fmt"

	"github.com/shopspring/decimal"
)

func MarshalDecimal(d decimal.Decimal) Marshaler {
	return MarshalString(d.String())
}

func UnmarshalDecimal(v any) (decimal.Decimal, error) {
	switch x := v.(type) {
	case string:
		return decimal.NewFromString(x)
	case int32:
		return decimal.NewFromInt32(x), nil
	case int64:
		return decimal.NewFromInt(x), nil
	case uint64:
		return decimal.NewFromUint64(x), nil
	case int:
		return decimal.NewFromInt(int64(x)), nil
	case float64:
		return decimal.NewFromFloat(x), nil
	case float32:
		return decimal.NewFromFloat32(x), nil
	default:
		return decimal.Zero, fmt.Errorf("%T is not a decimal", v)
	}
}
