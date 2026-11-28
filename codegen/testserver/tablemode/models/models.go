// Package models holds the Go types bound to the tablemode test schema. Both generated
// packages bind to them, so one set of resolvers serves both exec modes.
package models

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
)

type Node interface {
	IsNode()
	GetID() string
}

type Character interface {
	IsCharacter()
	IsNode()
	GetID() string
}

type SearchResult interface {
	IsSearchResult()
}

type Human struct {
	ID        string
	Name      string
	FriendIDs []string
	AppearsIn []Episode
	HeightM   float64
	Mass      *float64
	MassIn    *float64
	Secret    *string
	Tags      []*string
	Wallet    *Money
	Code      Code
	Words     Words
	TagList   Tags
	Counts    Counts
	WordsPtr  *Words
	MatrixA   Matrix
	MatrixB   Matrix
	nickname  string
}

// Matrix is a named list of lists that marshals itself, as the number of its rows.
type Matrix [][]string

func (m Matrix) MarshalGQL(out io.Writer) { graphql.MarshalInt(len(m)).MarshalGQL(out) }

func (m *Matrix) UnmarshalGQL(v any) error { return errors.New("matrix is output only") }

// GreetOrder is a method that takes its arguments in another order than the schema.
func (h *Human) GreetOrder(b, a string) string { return a + b }

// Tag is a named string that marshals itself, with a # in front.
type Tag string

func (t Tag) MarshalGQL(out io.Writer) { graphql.MarshalString("#" + string(t)).MarshalGQL(out) }

func (t *Tag) UnmarshalGQL(v any) error {
	s, err := graphql.UnmarshalString(v)
	*t = Tag(strings.TrimPrefix(s, "#"))
	return err
}

// Tags is a named slice of a type that marshals itself, which marshals itself too, as
// one string.
type Tags []Tag

func (ts Tags) MarshalGQL(out io.Writer) {
	s := make([]string, len(ts))
	for i, t := range ts {
		s[i] = string(t)
	}
	graphql.MarshalString(strings.Join(s, "+")).MarshalGQL(out)
}

func (ts *Tags) UnmarshalGQL(v any) error {
	list, ok := v.([]any)
	if !ok {
		return fmt.Errorf("tags must be a list, not %T", v)
	}
	for _, e := range list {
		s, ok := e.(string)
		if !ok {
			return fmt.Errorf("tags must be strings, not %T", e)
		}
		*ts = append(*ts, Tag("unmarshaled "+s))
	}
	return nil
}

// Counts is a named map type bound to a list of Int that marshals itself, as the number
// of its entries.
type Counts map[string]int

func (c Counts) MarshalGQL(out io.Writer) { graphql.MarshalInt(len(c)).MarshalGQL(out) }

func (c *Counts) UnmarshalGQL(v any) error {
	list, ok := v.([]any)
	if !ok {
		return fmt.Errorf("counts must be a list, not %T", v)
	}
	*c = Counts{}
	for i, e := range list {
		n, err := graphql.UnmarshalInt(e)
		if err != nil {
			return err
		}
		(*c)[strconv.Itoa(i)] = n
	}
	return nil
}

// MarshalBytes and UnmarshalBytes bind the Bytes scalar to []byte.
func MarshalBytes(b []byte) graphql.Marshaler { return graphql.MarshalString(string(b)) }

func UnmarshalBytes(v any) ([]byte, error) {
	s, err := graphql.UnmarshalString(v)
	return []byte(s), err
}

// Words is a named slice bound to a list of strings that marshals itself, as one string,
// and unmarshals the words of each string of the list.
type Words []string

func (w Words) MarshalGQL(out io.Writer) {
	graphql.MarshalString(strings.Join(w, " ")).MarshalGQL(out)
}

func (w *Words) UnmarshalGQL(v any) error {
	list, ok := v.([]any)
	if !ok {
		return fmt.Errorf("words must be a list, not %T", v)
	}
	for _, e := range list {
		s, ok := e.(string)
		if !ok {
			return fmt.Errorf("words must be strings, not %T", e)
		}
		*w = append(*w, strings.Fields(s)...)
	}
	return nil
}

func (*Human) IsNode()                {}
func (*Human) IsCharacter()           {}
func (*Human) IsSearchResult()        {}
func (h *Human) GetID() string        { return h.ID }
func (h *Human) SetNickname(n string) { h.nickname = n }

// Height is a method field with an argument.
func (h *Human) Height(unit LengthUnit) float64 {
	if unit == LengthUnitFoot {
		return h.HeightM * 3.28084
	}
	return h.HeightM
}

// Nickname is a method field of the (value, ok) shape.
func (h *Human) Nickname() (string, bool) {
	return h.nickname, h.nickname != ""
}

// Droid is bound as a value type, so interfaces see both Droid and *Droid.
type Droid struct {
	ID        string
	Name      string
	FriendIDs []string
	AppearsIn []Episode
	Function  string
	Serial    *int
}

func (Droid) IsNode()         {}
func (Droid) IsCharacter()    {}
func (Droid) IsSearchResult() {}
func (d Droid) GetID() string { return d.ID }

// PrimaryFunction takes a context, so the field resolves concurrently.
func (d Droid) PrimaryFunction(ctx context.Context) (string, error) {
	if d.Function == "" {
		return "", errors.New("droid has no function")
	}
	return d.Function, nil
}

type Starship struct {
	ID      string
	Name    string
	LengthM float64
	History [][]int
}

func (*Starship) IsSearchResult() {}

func (s *Starship) Length(unit LengthUnit) (float64, error) {
	if s.LengthM < 0 {
		return 0, errors.New("unknown length")
	}
	if unit == LengthUnitFoot {
		return s.LengthM * 3.28084, nil
	}
	return s.LengthM, nil
}

type Review struct {
	Stars      int
	Commentary *string
	Time       *time.Time
	Tags       []string
	Episode    *Episode
	Code       *Code
}

type NullChain struct {
	Required      *string
	Optional      *string
	Child         *NullChain
	RequiredChild *NullChain
}

// Query is the Go type of fields that return the query type.
type Query struct{}

type Summary struct {
	Label string
	Count int
	Extra map[string]any
}

type ReviewInput struct {
	Stars      int
	Commentary *string
	Time       *time.Time
	Tags       []string
	Nested     *NestedInput
	Note       *string
	Episode    *Episode
	Wallet     *Money
	Code       *Code
}

type NestedInput struct {
	Value   string
	Numbers []int
	Deeper  *NestedInput
}

// DefaultsInput is the input of the defaults schema, whose field has a Map default.
type DefaultsInput struct {
	Data map[string]any
}

type ResolvedInput struct {
	Label string
	Shout string
}

type OmittableInput struct {
	Value graphql.Omittable[*string]
	Count graphql.Omittable[*int]
}

// CodeBase is the model of the Code scalar. It has no marshal methods, so gqlgen would
// convert it from and to a string.
type CodeBase string

// Code is the Go type of the fields of type Code. It marshals itself, so gqlgen uses its
// methods although the model of the scalar is converted.
type Code CodeBase

func (c Code) MarshalGQL(w io.Writer) { graphql.MarshalString("#" + string(c)).MarshalGQL(w) }

func (c *Code) UnmarshalGQL(v any) error {
	s, ok := v.(string)
	if !ok || s == "" {
		return fmt.Errorf("code must be a non-empty string, got %v", v)
	}
	*c = Code(strings.TrimPrefix(s, "#"))
	return nil
}

// Money is a custom scalar that counts cents and is written as a decimal string.
type Money int64

func (m Money) MarshalGQL(w io.Writer) {
	_, _ = io.WriteString(w, strconv.Quote(fmt.Sprintf("%d.%02d", m/100, m%100)))
}

func (m *Money) UnmarshalGQL(v any) error {
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("money must be a string, got %T", v)
	}
	whole, frac, _ := strings.Cut(s, ".")
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid money %q", s)
	}
	f := int64(0)
	if frac != "" {
		if f, err = strconv.ParseInt(frac, 10, 64); err != nil || len(frac) != 2 {
			return fmt.Errorf("invalid money %q", s)
		}
	}
	*m = Money(w*100 + f)
	return nil
}

type Role string

const (
	RoleAdmin Role = "ADMIN"
	RoleUser  Role = "USER"
	RoleGuest Role = "GUEST"
)

func (e Role) IsValid() bool {
	switch e {
	case RoleAdmin, RoleUser, RoleGuest:
		return true
	}
	return false
}

func (e *Role) UnmarshalGQL(v any) error { return unmarshalEnum(e, v, "Role") }
func (e Role) MarshalGQL(w io.Writer)    { _, _ = io.WriteString(w, strconv.Quote(string(e))) }

type Episode string

const (
	EpisodeNewhope Episode = "NEWHOPE"
	EpisodeEmpire  Episode = "EMPIRE"
	EpisodeJedi    Episode = "JEDI"
)

func (e Episode) IsValid() bool {
	switch e {
	case EpisodeNewhope, EpisodeEmpire, EpisodeJedi:
		return true
	}
	return false
}

func (e *Episode) UnmarshalGQL(v any) error { return unmarshalEnum(e, v, "Episode") }
func (e Episode) MarshalGQL(w io.Writer)    { _, _ = io.WriteString(w, strconv.Quote(string(e))) }

// TagInput holds a scalar and an enum whose types carry a directive, and a Float with a
// default of more than six decimals.
type TagInput struct {
	Label    string
	Mood     *Mood
	Ratio    *float64
	Negative *float64
	Words    Words
}

// CaseArgs has a method whose parameters, id and ID, both take the argument id: gqlgen
// matches the names of parameters and arguments without regard to case.
type CaseArgs struct{}

func (*CaseArgs) F(
	id, ID string, //nolint:gocritic // the names differ in case only, as above
) string {
	return id + "," + ID
}

// MissingInput is the input that @missing is applied to.
type MissingInput struct {
	Value *string
}

// Bag is the second model of the object Bag, whose first model is map[string]any.
type Bag map[string]any

// BagHolder holds a Bag.
type BagHolder struct {
	Bag Bag
}

// ProbeInput is the input that @probe is applied to.
type ProbeInput struct {
	V *string
}

// NoteInput is the input that @note is applied to.
type NoteInput struct {
	V *string
	S *string
}

// RetypeInput holds fields whose Go types the functions mode and the runtime may write
// differently in errors: Extra is declared with interface{} rather than any.
type RetypeInput struct {
	Extra map[string]interface{} //nolint:revive // see above
	Blob  []byte
}

// ShapesInput holds the named slice and map types that unmarshal themselves.
type ShapesInput struct {
	TagList  Tags
	Counts   Counts
	WordsPtr *Words
}

// Fragile is marshaled by MarshalFragile, which panics for "boom" while the executor
// runs the field.
type Fragile string

func MarshalFragile(f Fragile) graphql.Marshaler {
	if f == "boom" {
		panic("fragile boom")
	}
	return graphql.MarshalString(string(f))
}

func UnmarshalFragile(v any) (Fragile, error) {
	s, err := graphql.UnmarshalString(v)
	return Fragile(s), err
}

// Tick is an event of the subscriptions. Fail fails for the tick numbered FailAt.
type Tick struct {
	N      int
	Label  string
	FailAt *int
}

func (t *Tick) Fail() (*string, error) {
	if t.FailAt != nil && *t.FailAt == t.N {
		return nil, fmt.Errorf("tick %d fails", t.N)
	}
	return new("ok"), nil
}

// Greet takes b before a, unlike the field in the schema.
func (h *Human) Greet(b, a Money) string {
	return h.Name + " " + strconv.FormatInt(int64(a), 10) + " " + strconv.FormatInt(int64(b), 10)
}

type Mood string

const (
	MoodHappy Mood = "HAPPY"
	MoodSad   Mood = "SAD"
)

func (e Mood) IsValid() bool { return e == MoodHappy || e == MoodSad }

func (e *Mood) UnmarshalGQL(v any) error { return unmarshalEnum(e, v, "Mood") }
func (e Mood) MarshalGQL(w io.Writer)    { _, _ = io.WriteString(w, strconv.Quote(string(e))) }

type LengthUnit string

const (
	LengthUnitMeter LengthUnit = "METER"
	LengthUnitFoot  LengthUnit = "FOOT"
)

func (e LengthUnit) IsValid() bool { return e == LengthUnitMeter || e == LengthUnitFoot }

func (e *LengthUnit) UnmarshalGQL(v any) error { return unmarshalEnum(e, v, "LengthUnit") }

func (e LengthUnit) MarshalGQL(
	w io.Writer,
) {
	_, _ = io.WriteString(w, strconv.Quote(string(e)))
}

type enum interface {
	~string
	IsValid() bool
}

func unmarshalEnum[E enum](e *E, v any, name string) error {
	s, ok := v.(string)
	if !ok {
		return errors.New("enums must be strings")
	}
	*e = E(s)
	if !(*e).IsValid() {
		return fmt.Errorf("%s is not a valid %s", s, name)
	}
	return nil
}
