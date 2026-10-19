// Package resolvers implements the tablemode test schema once for both generated
// packages. The generated resolver interfaces of the two packages have the same methods,
// so the types here satisfy both.
package resolvers

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/99designs/gqlgen/codegen/testserver/tablemode/models"
	"github.com/99designs/gqlgen/graphql"
)

var (
	humans = map[string]*models.Human{
		"1000": {
			ID: "1000", Name: "Luke", FriendIDs: []string{"1002", "2000", "2001"},
			AppearsIn: []models.Episode{models.EpisodeNewhope, models.EpisodeEmpire},
			HeightM:   1.72, Mass: new(77.0), MassIn: new(77.5), Secret: new("father"),
			Tags: []*string{new("jedi"), nil}, Wallet: new(models.Money(1234)), Code: "luke",
			Words:   models.Words{"use", "the", "force"},
			TagList: models.Tags{"jedi", "pilot"}, Counts: models.Counts{"a": 1, "b": 2},
			WordsPtr: &models.Words{"may", "the"},
			MatrixA:  models.Matrix{{"a"}}, MatrixB: models.Matrix{{"1"}, {"2"}},
		},
		"1001": {
			ID: "1001", Name: "Vader", AppearsIn: []models.Episode{models.EpisodeEmpire},
			HeightM: 2.02,
		},
		"1002": {
			ID: "1002", Name: "Han", FriendIDs: []string{"1000"},
			AppearsIn: []models.Episode{models.EpisodeJedi}, HeightM: 1.8,
		},
	}
	droids = map[string]models.Droid{
		"2000": {
			ID:        "2000",
			Name:      "C-3PO",
			Function:  "Protocol",
			FriendIDs: []string{"1000"},
			Serial:    new(3),
		},
		"2001": {ID: "2001", Name: "R2-D2", FriendIDs: []string{"2000"}},
	}
	starships = map[string]*models.Starship{
		"3000": {ID: "3000", Name: "Falcon", LengthM: 34.37, History: [][]int{{1, 2}, {3}}},
		"3001": {ID: "3001", Name: "X-Wing", LengthM: -1, History: [][]int{}},
	}
)

func init() {
	humans["1002"].SetNickname("Solo")
}

func character(id string) models.Character {
	if h, ok := humans[id]; ok {
		return h
	}
	if d, ok := droids[id]; ok {
		// A value Droid exercises the non-pointer case of the interface marshalers.
		return d
	}
	return nil
}

func friends(ids []string) []models.Character {
	res := make([]models.Character, 0, len(ids))
	for _, id := range ids {
		res = append(res, character(id))
	}
	return res
}

type Query struct{}

func (Query) Hero(ctx context.Context, episode *models.Episode) (models.Character, error) {
	if episode != nil && *episode == models.EpisodeEmpire {
		return humans["1000"], nil
	}
	if episode != nil && *episode == models.EpisodeNewhope {
		return nil, nil
	}
	return droids["2001"], nil
}

func (Query) Human(ctx context.Context, id string) (*models.Human, error) {
	return humans[id], nil
}

func (Query) Droid(ctx context.Context, id string) (*models.Droid, error) {
	d, ok := droids[id]
	if !ok {
		return nil, fmt.Errorf("no droid %s", id)
	}
	return &d, nil
}

func (Query) Node(ctx context.Context, id string) (models.Node, error) {
	if c := character(id); c != nil {
		return c, nil
	}
	return nil, nil
}

func (Query) Search(ctx context.Context, text string) ([]models.SearchResult, error) {
	var res []models.SearchResult
	ids := make([]string, 0, len(humans)+len(droids)+len(starships))
	for id := range humans {
		ids = append(ids, id)
	}
	for id := range droids {
		ids = append(ids, id)
	}
	for id := range starships {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		switch {
		case humans[id] != nil && strings.Contains(humans[id].Name, text):
			res = append(res, humans[id])
		case droids[id].ID != "" && strings.Contains(droids[id].Name, text):
			d := droids[id]
			res = append(res, &d)
		case starships[id] != nil && strings.Contains(starships[id].Name, text):
			res = append(res, starships[id])
		}
	}
	return res, nil
}

func (Query) Starship(ctx context.Context, id string) (*models.Starship, error) {
	return starships[id], nil
}

func (Query) Reviews(
	ctx context.Context,
	episode models.Episode,
	since *time.Time,
) ([]*models.Review, error) {
	t := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	reviews := []*models.Review{
		{Stars: 5, Commentary: new("great"), Time: &t, Tags: []string{"a"}, Episode: &episode},
		{Stars: 2, Tags: []string{}},
	}
	if since != nil && since.After(t) {
		return reviews[1:], nil
	}
	return reviews, nil
}

func review(in models.ReviewInput) *models.Review {
	r := &models.Review{
		Stars:      in.Stars,
		Commentary: in.Commentary,
		Time:       in.Time,
		Tags:       in.Tags,
		Episode:    in.Episode,
		Code:       in.Code,
	}
	if r.Tags == nil {
		r.Tags = []string{}
	}
	if in.Nested != nil {
		for n := in.Nested; n != nil; n = n.Deeper {
			r.Tags = append(r.Tags, n.Value)
			for _, i := range n.Numbers {
				r.Tags = append(r.Tags, strconv.Itoa(i))
			}
		}
	}
	if in.Note != nil {
		r.Tags = append(r.Tags, "note:"+*in.Note)
	}
	if in.Wallet != nil {
		r.Tags = append(r.Tags, fmt.Sprintf("wallet:%d", *in.Wallet))
	}
	return r
}

func (Query) EchoReview(ctx context.Context, input models.ReviewInput) (*models.Review, error) {
	return review(input), nil
}

func (Query) EchoReviews(
	ctx context.Context,
	inputs []*models.ReviewInput,
) ([]*models.Review, error) {
	if inputs == nil {
		return nil, nil
	}
	res := make([]*models.Review, 0, len(inputs))
	for _, in := range inputs {
		res = append(res, review(*in))
	}
	return res, nil
}

func (Query) EchoMap(ctx context.Context, input map[string]any) (map[string]any, error) {
	if input == nil {
		return map[string]any{"empty": true}, nil
	}
	res := map[string]any{}
	for k, v := range input {
		res[k] = describe(v)
	}
	return res, nil
}

func describe(v any) string {
	switch v := v.(type) {
	case *string:
		if v != nil {
			return "*string:" + *v
		}
	case *int:
		if v != nil {
			return fmt.Sprintf("*int:%d", *v)
		}
	case *models.Role:
		if v != nil {
			return "*Role:" + string(*v)
		}
	}
	return fmt.Sprintf("%T:%v", v, v)
}

func (Query) EchoResolved(ctx context.Context, input models.ResolvedInput) (string, error) {
	return input.Label + "/" + input.Shout, nil
}

func (Query) EchoOmittable(ctx context.Context, input models.OmittableInput) (string, error) {
	describe := func(set bool, v any) string {
		if !set {
			return "unset"
		}
		return fmt.Sprintf("%v", v)
	}
	value := "nil"
	if v := input.Value.Value(); v != nil {
		value = *v
	}
	count := "nil"
	if c := input.Count.Value(); c != nil {
		count = strconv.Itoa(*c)
	}
	return describe(input.Value.IsSet(), value) + "," + describe(input.Count.IsSet(), count), nil
}

func (Query) EchoNullified(ctx context.Context, value *string) (string, error) {
	if value == nil {
		return "nil", nil
	}
	return *value, nil
}

func (Query) Money(ctx context.Context, amount models.Money) (models.Money, error) {
	return amount * 2, nil
}

func (Query) Roles(ctx context.Context) ([]models.Role, error) {
	return []models.Role{models.RoleAdmin, models.RoleGuest}, nil
}

func (Query) WithDefaults(
	ctx context.Context,
	a *int,
	b *string,
	c []models.Episode,
	d *models.NestedInput,
) (string, error) {
	parts := []string{}
	if a != nil {
		parts = append(parts, strconv.Itoa(*a))
	}
	if b != nil {
		parts = append(parts, *b)
	}
	parts = append(parts, fmt.Sprint(c))
	if d != nil {
		parts = append(parts, d.Value, fmt.Sprint(d.Numbers))
	}
	return strings.Join(parts, "|"), nil
}

// Fragile returns value as a Fragile, whose marshaler panics for "boom".
func (Query) Fragile(ctx context.Context, value string) (*models.Fragile, error) {
	f := models.Fragile(value)
	return &f, nil
}

func (Query) FragileRequired(ctx context.Context, value string) (models.Fragile, error) {
	return models.Fragile(value), nil
}

func (Query) Fragiles(ctx context.Context, values []string) ([]models.Fragile, error) {
	res := make([]models.Fragile, len(values))
	for i, v := range values {
		res[i] = models.Fragile(v)
	}
	return res, nil
}

// special returns NaN, +Inf or -Inf for those kinds, and 1.5 for others.
func special(kind string) float64 {
	switch kind {
	case "nan":
		return math.NaN()
	case "inf":
		return math.Inf(1)
	case "-inf":
		return math.Inf(-1)
	}
	return 1.5
}

func (Query) Special(ctx context.Context, kind string) (*float64, error) {
	return new(special(kind)), nil
}

func (Query) SpecialRequired(ctx context.Context, kind string) (float64, error) {
	return special(kind), nil
}

func (Query) Exploding(ctx context.Context) (*string, error) {
	return new("never reached"), nil
}

// DecodeInputs decodes raw into each input type through the index of input
// unmarshalers of the request, as a resolver that calls the generated UnmarshalXInput
// does, and answers with the values and errors.
func (Query) DecodeInputs(ctx context.Context, raw map[string]any) (string, error) {
	var parts []string
	decode := func(name string, out any) {
		err := graphql.UnmarshalNamedInputFromContext(ctx, name, raw, out)
		parts = append(parts, name+"="+Describe(out)+" err="+fmt.Sprint(err))
	}
	var review models.ReviewInput
	decode("ReviewInput", &review)
	var reviewPtr *models.ReviewInput
	decode("ReviewInput", &reviewPtr)
	var nested models.NestedInput
	decode("NestedInput", &nested)
	var m map[string]any
	decode("MapInput", &m)
	var omittable models.OmittableInput
	decode("OmittableInput", &omittable)
	var resolved models.ResolvedInput
	decode("ResolvedInput", &resolved)
	var tag models.TagInput
	decode("TagInput", &tag)
	return strings.Join(parts, " | "), nil
}

type Subscription struct{}

// stream sends the events that event makes for 1 to count, at most five, and closes the
// channel.
func stream[T any](ctx context.Context, count int, event func(i int) T) <-chan T {
	n := min(max(count, 0), 5)
	ch := make(chan T)
	go func() {
		defer close(ch)
		for i := 1; i <= n; i++ {
			select {
			case ch <- event(i):
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch
}

// tick returns the tick numbered i, or nil when i is nullAt.
func tick(i int, failAt, nullAt *int) *models.Tick {
	if nullAt != nil && *nullAt == i {
		return nil
	}
	return &models.Tick{N: i, Label: "tick " + strconv.Itoa(i), FailAt: failAt}
}

func (Subscription) Ticks(
	ctx context.Context,
	count int,
	failAt *int,
	nullAt *int,
) (<-chan *models.Tick, error) {
	return stream(ctx, count, func(i int) *models.Tick { return tick(i, failAt, nullAt) }), nil
}

func (Subscription) RequiredTicks(
	ctx context.Context,
	count int,
	nullAt *int,
) (<-chan *models.Tick, error) {
	return stream(ctx, count, func(i int) *models.Tick { return tick(i, nil, nullAt) }), nil
}

func (Subscription) TickBatches(ctx context.Context, count int) (<-chan []*models.Tick, error) {
	return stream(ctx, count, func(i int) []*models.Tick {
		ticks := make([]*models.Tick, i)
		for j := range ticks {
			ticks[j] = tick(j+1, nil, nil)
		}
		return ticks
	}), nil
}

func (Subscription) FailingStream(ctx context.Context) (<-chan *int, error) {
	return nil, errors.New("stream fails")
}

func (Subscription) PanickingStream(ctx context.Context) (<-chan *int, error) {
	panic("stream panics")
}

func (Subscription) SecretTicks(ctx context.Context) (<-chan *int, error) {
	return stream(ctx, 1, func(i int) *int { return &i }), nil
}

func (Query) Trimmed(ctx context.Context, text string) (string, error) {
	return strconv.Quote(text), nil
}

func (Query) NulledRatio(ctx context.Context) (*string, error) { return new("nulled"), nil }

func (Query) NulledNonNull(ctx context.Context) (*string, error) { return new("nulled"), nil }

func (Query) BagHolder(ctx context.Context) (*models.BagHolder, error) {
	return &models.BagHolder{Bag: models.Bag{"name": "bag"}}, nil
}

func (Query) MissingField(ctx context.Context) (*int, error) { return new(1), nil }

func (Query) MissingArg(ctx context.Context, x *string) (*string, error) { return x, nil }

func (Query) MissingInput(ctx context.Context, input *models.MissingInput) (*string, error) {
	return new(Describe(input)), nil
}

func (Query) PluginGuarded(ctx context.Context) (*string, error) { return new("unguarded"), nil }

func (Query) PluginTrimmed(ctx context.Context, text string) (string, error) {
	return strconv.Quote(text), nil
}

// PluginRequired fails, so that its null propagates as the plugin made it non-null.
func (Query) PluginRequired(ctx context.Context) (string, error) {
	return "", errors.New("required fails")
}

// PluginRemoved is not called: the plugin of the generator removes the field.
func (Query) PluginRemoved(ctx context.Context) (*string, error) { return new("removed"), nil }

func (Query) Probed(ctx context.Context, x *string, input *models.ProbeInput) (*string, error) {
	return new(Describe(x) + " " + Describe(input)), nil
}

func (Query) Noted(ctx context.Context, x *string, input *models.NoteInput) (*string, error) {
	return new(Describe(x) + " " + Describe(input)), nil
}

func (Query) Retyped(ctx context.Context, blob []byte) (*string, error) {
	return new(Describe(blob)), nil
}

func (Query) RetypedInput(ctx context.Context, input models.RetypeInput) (*string, error) {
	return new(Describe(input)), nil
}

func (Query) RetypedField(ctx context.Context) ([]byte, error) { return []byte("raw"), nil }

func (Query) Fleets(ctx context.Context) ([][]*models.Starship, error) {
	return [][]*models.Starship{
		{starships["3000"], starships["3001"]},
		{starships["3001"]},
	}, nil
}

func (Query) Fleet(ctx context.Context) ([]*models.Starship, error) {
	return []*models.Starship{starships["3000"], starships["3001"], starships["3000"], nil}, nil
}

// Stubbed fails with the error of a directive without an implementation, as a resolver
// that is not written yet might.
func (Query) Stubbed(ctx context.Context) (*string, error) {
	return nil, errors.New("stub: directive upper is not implemented")
}

func (Query) Shapes(ctx context.Context, input models.ShapesInput) (*string, error) {
	return new(Describe(input)), nil
}

// Tag answers with what it received, so that the responses show the values of the input
// and of the Float defaults.
func (Query) Tag(ctx context.Context, input models.TagInput, eps *float64) (string, error) {
	return Describe(input) + " eps=" + Describe(eps), nil
}

func (Query) NullChain(ctx context.Context) (*models.NullChain, error) {
	return &models.NullChain{
		Required: new("root"),
		Child: &models.NullChain{
			Optional:      new("child"),
			RequiredChild: &models.NullChain{Required: new("grandchild")},
		},
	}, nil
}

func (Query) Failing(ctx context.Context) (string, error) {
	return "", errors.New("failing on purpose")
}

func (Query) FailingOptional(ctx context.Context) (*string, error) {
	return nil, errors.New("optional failure")
}

func (Query) Panicking(ctx context.Context) (*string, error) {
	panic("panicking on purpose")
}

func (Query) Summary(ctx context.Context) (*models.Summary, error) {
	return &models.Summary{Label: "summary", Count: 3, Extra: map[string]any{"k": "v"}}, nil
}

func (Query) SecretNumber(ctx context.Context) (*int, error) {
	return new(42), nil
}

type Mutation struct{}

func (Mutation) CreateReview(
	ctx context.Context,
	episode models.Episode,
	input models.ReviewInput,
) (*models.Review, error) {
	r := review(input)
	r.Episode = &episode
	return r, nil
}

func (Mutation) SetRole(ctx context.Context, role models.Role) (models.Role, error) {
	return role, nil
}

type Human struct{}

func (Human) Friends(ctx context.Context, obj *models.Human) ([]models.Character, error) {
	RecordingOf(ctx).Add("resolver Human.friends obj=%s", Describe(obj))
	return friends(obj.FriendIDs), nil
}

func (Human) Starships(ctx context.Context, obj *models.Human) ([]*models.Starship, error) {
	RecordingOf(ctx).Add("resolver Human.starships obj=%s", Describe(obj))
	if obj.ID == "1001" {
		return nil, errors.New("starships are classified")
	}
	return []*models.Starship{starships["3000"], starships["3001"]}, nil
}

type Droid struct{}

func (Droid) Friends(ctx context.Context, obj *models.Droid) ([]models.Character, error) {
	RecordingOf(ctx).Add("resolver Droid.friends obj=%s", Describe(obj))
	return friends(obj.FriendIDs), nil
}

// starshipIDs returns the ids of objs, which the batch resolvers record. The parents of
// a list hold its nil elements too.
func starshipIDs(objs []*models.Starship) string {
	ids := make([]string, len(objs))
	for i, s := range objs {
		ids[i] = "nil"
		if s != nil {
			ids[i] = s.ID
		}
	}
	return strings.Join(ids, ",")
}

// LastSeen fails for all the droids of objs when R2-D2 is one of them.
func (Droid) LastSeen(ctx context.Context, objs []*models.Droid) ([]*string, error) {
	ids := make([]string, len(objs))
	res := make([]*string, len(objs))
	for i, d := range objs {
		ids[i] = d.ID
		res[i] = new(d.Name + " on Tatooine")
	}
	RecordingOf(ctx).Add("batch Droid.lastSeen objs=%s", strings.Join(ids, ","))
	for _, d := range objs {
		if d.ID == "2001" {
			return nil, errors.New("R2-D2 is hiding")
		}
	}
	return res, nil
}

type Starship struct{}

func (Starship) Crew(ctx context.Context, objs []*models.Starship) ([]*int, error) {
	RecordingOf(ctx).Add("batch Starship.crew objs=%s", starshipIDs(objs))
	res := make([]*int, len(objs))
	for i, s := range objs {
		if s != nil && s.LengthM >= 0 {
			res[i] = new(int(s.LengthM) / 10)
		}
	}
	return res, nil
}

// Rating fails for the starships of unknown length, with an error of their own.
func (Starship) Rating(
	ctx context.Context,
	objs []*models.Starship,
	scale *int,
) ([]float64, error) {
	RecordingOf(ctx).Add(
		"batch Starship.rating objs=%s scale=%s",
		starshipIDs(objs),
		Describe(scale),
	)
	n := 5
	if scale != nil {
		n = *scale
	}
	res := make([]float64, len(objs))
	errs := make(graphql.BatchErrorList, len(objs))
	failed := false
	for i, s := range objs {
		if s == nil {
			continue
		}
		if s.LengthM < 0 {
			errs[i] = fmt.Errorf("%s has no rating", s.Name)
			failed = true
			continue
		}
		res[i] = s.LengthM * float64(n) / 100
	}
	if failed {
		return res, errs
	}
	return res, nil
}

// Miscount returns one result more than there are starships.
func (Starship) Miscount(ctx context.Context, objs []*models.Starship) ([]*int, error) {
	RecordingOf(ctx).Add("batch Starship.miscount objs=%s", starshipIDs(objs))
	return make([]*int, len(objs)+1), nil
}

// Query is never called: a field that returns the query type resolves to an empty value of
// the type.
func (Starship) Query(ctx context.Context, objs []*models.Starship) ([]*models.Query, error) {
	return nil, errors.New("the batch resolver of Starship.query was called")
}

type ResolvedInput struct{}

func (ResolvedInput) Shout(ctx context.Context, obj *models.ResolvedInput, data *string) error {
	RecordingOf(ctx).Add("resolver ResolvedInput.shout obj=%s data=%s",
		Describe(obj), Describe(data))
	if data == nil {
		obj.Shout = "quiet"
		return nil
	}
	if *data == "" {
		return errors.New("shout must not be empty")
	}
	obj.Shout = strings.ToUpper(*data)
	return nil
}

func Upper(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	res, err := next(ctx)
	RecordingOf(ctx).Add("directive upper obj=%s -> %s err=%v", Describe(obj), Describe(res), err)
	if s, ok := res.(string); ok {
		return strings.ToUpper(s), err
	}
	return res, err
}

func Auth(ctx context.Context, obj any, next graphql.Resolver, role models.Role) (any, error) {
	RecordingOf(ctx).Add("directive auth obj=%s role=%s", Describe(obj), Describe(role))
	if role == models.RoleAdmin {
		return nil, fmt.Errorf("role %s required for %T", role, obj)
	}
	return next(ctx)
}

func Length(
	ctx context.Context,
	obj any,
	next graphql.Resolver,
	minimum int,
	maximum *int,
) (any, error) {
	res, err := next(ctx)
	RecordingOf(ctx).Add("directive length obj=%s min=%d max=%s -> %s err=%v",
		Describe(obj), minimum, Describe(maximum), Describe(res), err)
	if err != nil {
		return res, err
	}
	var s string
	switch v := res.(type) {
	case string:
		s = v
	case *string:
		if v == nil {
			return res, nil
		}
		s = *v
	default:
		return nil, fmt.Errorf("length cannot check %T", res)
	}
	if len(s) < minimum || (maximum != nil && len(s) > *maximum) {
		return nil, fmt.Errorf("length of %q is out of range", s)
	}
	return res, nil
}

// TrimArg trims the strings in obj, the raw arguments of the field, and passes on.
func TrimArg(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	if m, ok := obj.(map[string]any); ok {
		for k, v := range m {
			if s, ok := v.(string); ok {
				m[k] = strings.TrimSpace(s)
			}
		}
	}
	res, err := next(ctx)
	RecordingOf(ctx).Add("directive trimArg -> %s err=%v", Describe(res), err)
	return res, err
}

// FromContext sets the argument authorID when the request leaves it out, as a directive
// that takes it from the context would.
func FromContext(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	if m, ok := obj.(map[string]any); ok {
		if _, ok := m["authorID"]; !ok {
			m["authorID"] = "u1"
		}
	}
	return next(ctx)
}

// Mark returns a directive that replaces the value with the name of the directive.
func Mark(name string) func(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	return func(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
		if _, err := next(ctx); err != nil {
			return nil, err
		}
		return new("marked by @" + name), nil
	}
}

// Note records its text and passes the value on.
func Note(ctx context.Context, obj any, next graphql.Resolver, text *string) (any, error) {
	RecordingOf(ctx).Add("directive note text=%s obj=%s", Describe(text), Describe(obj))
	return next(ctx)
}

// Nulled records its arguments and passes the value on.
func Nulled(
	ctx context.Context,
	obj any,
	next graphql.Resolver,
	names []string,
	data map[string]any,
) (any, error) {
	RecordingOf(ctx).Add("directive nulled names=%s data=%s", Describe(names), Describe(data))
	return next(ctx)
}

// Probe records the object it receives and its arguments.
func Probe(
	ctx context.Context,
	obj any,
	next graphql.Resolver,
	rawArgs *string,
	asMap *string,
	it *string,
	rawArgsArg *string,
	nilArg *string,
	trueArg *bool,
	on *bool,
) (any, error) {
	RecordingOf(ctx).Add(
		"directive probe obj=%s rawArgs=%s asMap=%s it=%s rawArgsArg=%s nil=%s true=%s on=%s",
		Describe(obj), Describe(rawArgs), Describe(asMap), Describe(it), Describe(rawArgsArg),
		Describe(nilArg), Describe(trueArg), Describe(on))
	return next(ctx)
}

// Seen records the Go types of the object it receives and of the value, and passes the
// value on.
func Seen(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	res, err := next(ctx)
	RecordingOf(ctx).Add("directive seen obj=%T -> %T err=%v", obj, res, err)
	return res, err
}

// Order adds name to obj, the raw arguments of the field, and records the names so far.
func Order(ctx context.Context, obj any, next graphql.Resolver, name string) (any, error) {
	if m, ok := obj.(map[string]any); ok {
		seen, _ := m["_order"].(string)
		m["_order"] = seen + name
		RecordingOf(ctx).Add("directive order seen=%s", m["_order"])
	}
	return next(ctx)
}

// Retype returns a string in place of the value.
func Retype(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	res, err := next(ctx)
	RecordingOf(ctx).Add("directive retype -> %s err=%v", Describe(res), err)
	if err != nil {
		return res, err
	}
	return "retyped", nil
}

// Explode panics, to compare how the modes recover from a directive that panics.
func Explode(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	RecordingOf(ctx).Add("directive explode obj=%s", Describe(obj))
	panic("explode")
}

func LogSubscription(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	RecordingOf(ctx).Add("directive logSubscription obj=%s", Describe(obj))
	return next(ctx)
}

// Tagged records what it receives, including its arguments, and passes the value on.
func Tagged(
	ctx context.Context,
	obj any,
	next graphql.Resolver,
	note *string,
	ratio *float64,
) (any, error) {
	res, err := next(ctx)
	RecordingOf(ctx).Add("directive tagged obj=%s note=%s ratio=%s -> %s err=%v",
		Describe(obj), Describe(note), Describe(ratio), Describe(res), err)
	return res, err
}

// ArgNote records that it ran. The schema applies it only to an argument of the
// definition of @tagged.
func ArgNote(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	res, err := next(ctx)
	RecordingOf(ctx).Add("directive argNote obj=%s -> %s err=%v", Describe(obj), Describe(res), err)
	return res, err
}

func Nullify(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	res, err := next(ctx)
	RecordingOf(ctx).Add("directive nullify obj=%s -> %s err=%v", Describe(obj), Describe(res), err)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func ValidReview(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	res, err := next(ctx)
	RecordingOf(ctx).Add("directive validReview obj=%s -> %s err=%v",
		Describe(obj), Describe(res), err)
	if err != nil {
		return res, err
	}
	var stars int
	switch v := res.(type) {
	case models.ReviewInput:
		stars = v.Stars
		v.Tags = append(v.Tags, "validated")
		res = v
	case *models.ReviewInput:
		stars = v.Stars
		v.Tags = append(v.Tags, "validated")
	}
	if stars > 5 {
		return nil, errors.New("a review has at most 5 stars")
	}
	return res, nil
}

func Trace(ctx context.Context, obj any, next graphql.Resolver, label *string) (any, error) {
	res, err := next(ctx)
	RecordingOf(ctx).Add("directive trace obj=%s label=%s -> %s err=%v",
		Describe(obj), Describe(label), Describe(res), err)
	if s, ok := res.(string); ok && label != nil {
		return s + " [" + *label + "]", err
	}
	return res, err
}

func LogQuery(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	RecordingOf(ctx).Add("directive logQuery obj=%s", Describe(obj))
	return next(ctx)
}

func LogMutation(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	RecordingOf(ctx).Add("directive logMutation obj=%s", Describe(obj))
	return next(ctx)
}

// HeightComplexity makes heights in feet expensive, to test that complexity sees the
// arguments.
func HeightComplexity(childComplexity int, unit models.LengthUnit) int {
	if unit == models.LengthUnitFoot {
		return 100
	}
	return 1
}
