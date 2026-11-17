package graphql

import (
	"io"
	"strconv"
	"testing"
)

// FieldSet.MarshalGQL runs once for every object in a response, so whatever it
// allocates is multiplied by the number of objects returned.
func BenchmarkFieldSetMarshalGQL(b *testing.B) {
	for _, n := range []int{4, 16} {
		b.Run("fields="+strconv.Itoa(n), func(b *testing.B) {
			fields := make([]CollectedField, n)
			for i := range fields {
				fields[i] = collectedField("field" + strconv.Itoa(i))
			}
			out := NewFieldSet(fields)
			for i := range out.Values {
				out.Values[i] = MarshalString("value")
			}

			b.ReportAllocs()
			for b.Loop() {
				out.MarshalGQL(io.Discard)
			}
		})
	}
}
