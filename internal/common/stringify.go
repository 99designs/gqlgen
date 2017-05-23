package common

import (
	"fmt"
	"sort"
	"strings"

	"github.com/neelance/graphql-go/internal/lexer"
)

func Stringify(v interface{}) string {
	switch v := v.(type) {
	case *lexer.BasicLit:
		return v.Text

	case []interface{}:
		entries := make([]string, len(v))
		for i, entry := range v {
			entries[i] = Stringify(entry)
		}
		return "[" + strings.Join(entries, ", ") + "]"

	case map[string]interface{}:
		names := make([]string, 0, len(v))
		for name := range v {
			names = append(names, name)
		}
		sort.Strings(names)

		entries := make([]string, 0, len(names))
		for _, name := range names {
			entries = append(entries, name+": "+Stringify(v[name]))
		}
		return "{" + strings.Join(entries, ", ") + "}"

	case nil:
		return "null"

	default:
		return fmt.Sprintf("(invalid type: %T)", v)
	}
}
