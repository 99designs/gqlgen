package resolvers

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

// Recording holds what the resolvers, the directives and the middleware of a request
// receive, so that a test can compare the Go values the two modes pass, which the
// responses do not show: a nil of a type against nil itself, a nil slice against an
// empty one, or an unset Omittable against a set one.
type Recording struct {
	mu    sync.Mutex
	lines []string
	// values are the arguments the request handed to the resolvers. Mark changes them
	// after the request, so that a value shared with a later request shows there.
	values []any
}

type recordingKey struct{}

// WithRecording returns ctx carrying r, for the resolvers and directives to record into.
func WithRecording(ctx context.Context, r *Recording) context.Context {
	return context.WithValue(ctx, recordingKey{}, r)
}

// RecordingOf returns the Recording of ctx, or nil when the request records nothing.
func RecordingOf(ctx context.Context) *Recording {
	r, _ := ctx.Value(recordingKey{}).(*Recording)
	return r
}

// Add records a line. A nil Recording records nothing.
func (tr *Recording) Add(format string, args ...any) {
	if tr == nil {
		return
	}
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.lines = append(tr.lines, fmt.Sprintf(format, args...))
}

// Keep remembers v for Mark.
func (tr *Recording) Keep(v any) {
	if tr == nil {
		return
	}
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.values = append(tr.values, v)
}

// Lines returns the lines recorded so far, sorted, as the fields of a request resolve
// in an order that depends on timing.
func (tr *Recording) Lines() []string {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	lines := slices.Clone(tr.lines)
	sort.Strings(lines)
	return lines
}

// Mark changes the maps, slices and structs the request handed to the resolvers: it
// adds a key to every map, and sets the first element of every slice and every string
// field of every struct to "mutated". A later request that is given the same value
// shows the marks.
func (tr *Recording) Mark() {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	for _, v := range tr.values {
		mark(reflect.ValueOf(v), 0)
	}
}

func mark(v reflect.Value, depth int) {
	if depth > 4 || !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Interface, reflect.Pointer:
		if !v.IsNil() {
			mark(v.Elem(), depth+1)
		}
	case reflect.Map:
		if v.IsNil() {
			return
		}
		if v.Type().Key().Kind() == reflect.String && v.Type().Elem().Kind() == reflect.Interface {
			v.SetMapIndex(reflect.ValueOf("_mutated"), reflect.ValueOf(true))
		}
		for _, k := range v.MapKeys() {
			mark(v.MapIndex(k), depth+1)
		}
	case reflect.Slice:
		for i := range v.Len() {
			e := v.Index(i)
			switch {
			case i == 0 && e.Kind() == reflect.String && e.CanSet():
				e.SetString("mutated")
			case i == 0 && e.Kind() == reflect.Interface && e.CanSet():
				e.Set(reflect.ValueOf("mutated"))
			default:
				mark(e, depth+1)
			}
		}
	case reflect.Struct:
		for i := range v.NumField() {
			f := v.Field(i)
			if !v.Type().Field(i).IsExported() {
				continue
			}
			if f.Kind() == reflect.String && f.CanSet() {
				f.SetString("mutated")
			} else {
				mark(f, depth+1)
			}
		}
	}
}

// Describe writes v so that two values of the same shape are written the same, and
// values the responses do not tell apart are not: a nil pointer is "(*T)(nil)" and nil
// itself "nil", a nil slice "[]T(nil)" and an empty one "[]T{}", an Omittable says
// whether it is set, and the keys of a map are in order.
func Describe(v any) string {
	return describeValue(reflect.ValueOf(v), 0)
}

func describeValue(v reflect.Value, depth int) string {
	if !v.IsValid() {
		return "nil"
	}
	t := v.Type()
	if depth > 5 {
		return t.String() + "{…}"
	}
	switch v.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			return t.String() + "(nil)"
		}
		return describeValue(v.Elem(), depth)
	case reflect.Pointer:
		if v.IsNil() {
			return "(" + t.String() + ")(nil)"
		}
		return "&" + describeValue(v.Elem(), depth+1)
	case reflect.Slice:
		if v.IsNil() {
			return t.String() + "(nil)"
		}
		fallthrough
	case reflect.Array:
		parts := make([]string, 0, v.Len())
		for i := range v.Len() {
			if i == 8 {
				parts = append(parts, fmt.Sprintf("… %d more", v.Len()-i))
				break
			}
			parts = append(parts, describeValue(v.Index(i), depth+1))
		}
		return t.String() + "{" + strings.Join(parts, ", ") + "}"
	case reflect.Map:
		if v.IsNil() {
			return t.String() + "(nil)"
		}
		keys := v.MapKeys()
		sort.Slice(keys, func(i, j int) bool {
			return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface())
		})
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			value := describeValue(v.MapIndex(k), depth+1)
			parts = append(parts, fmt.Sprintf("%v: %s", k.Interface(), value))
		}
		return t.String() + "{" + strings.Join(parts, ", ") + "}"
	case reflect.Struct:
		if t == reflect.TypeFor[time.Time]() {
			return "time.Time(" + v.Interface().(time.Time).UTC().Format(time.RFC3339Nano) + ")"
		}
		if strings.HasPrefix(t.Name(), "Omittable[") {
			if !v.MethodByName("IsSet").Call(nil)[0].Bool() {
				return "Omittable(unset)"
			}
			return "Omittable(" + describeValue(v.MethodByName("Value").Call(nil)[0], depth+1) + ")"
		}
		parts := make([]string, 0, v.NumField())
		for i := range v.NumField() {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			parts = append(parts, f.Name+": "+describeValue(v.Field(i), depth+1))
		}
		return t.String() + "{" + strings.Join(parts, ", ") + "}"
	case reflect.Func, reflect.Chan:
		return t.String()
	case reflect.String:
		return fmt.Sprintf("%s(%q)", t.String(), v.String())
	default:
		return fmt.Sprintf("%s(%v)", t.String(), v)
	}
}
