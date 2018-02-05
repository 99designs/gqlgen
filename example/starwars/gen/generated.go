package gen

import (
	"context"
	"fmt"
	"github.com/vektah/graphql-go/example/starwars"
	"github.com/vektah/graphql-go/introspection"
	"github.com/vektah/graphql-go/query"
	"github.com/vektah/graphql-go/schema"
	"strconv"
)

type Resolvers interface {
	Droid_friends(ctx context.Context, it *starwars.Droid) ([]starwars.Character, error)
	Droid_friendsConnection(ctx context.Context, it *starwars.Droid, first *int, after *string) (starwars.FriendsConnection, error)
	Human_friends(ctx context.Context, it *starwars.Human) ([]starwars.Character, error)
	Human_friendsConnection(ctx context.Context, it *starwars.Human, first *int, after *string) (starwars.FriendsConnection, error)
	Human_starships(ctx context.Context, it *starwars.Human) ([]starwars.Starship, error)
	Mutation_createReview(ctx context.Context, episode string, review starwars.Review) (*starwars.Review, error)
	Query_hero(ctx context.Context, episode *string) (starwars.Character, error)
	Query_reviews(ctx context.Context, episode string) ([]starwars.Review, error)
	Query_search(ctx context.Context, text string) ([]starwars.SearchResult, error)
	Query_character(ctx context.Context, id string) (starwars.Character, error)
	Query_droid(ctx context.Context, id string) (*starwars.Droid, error)
	Query_human(ctx context.Context, id string) (*starwars.Human, error)
	Query_starship(ctx context.Context, id string) (*starwars.Starship, error)
}

var (
	droidSatisfies             = []string{"Droid", "Character"}
	friendsConnectionSatisfies = []string{"FriendsConnection"}
	friendsEdgeSatisfies       = []string{"FriendsEdge"}
	humanSatisfies             = []string{"Human", "Character"}
	mutationSatisfies          = []string{"Mutation"}
	pageInfoSatisfies          = []string{"PageInfo"}
	querySatisfies             = []string{"Query"}
	reviewSatisfies            = []string{"Review"}
	starshipSatisfies          = []string{"Starship"}
	__DirectiveSatisfies       = []string{"__Directive"}
	__EnumValueSatisfies       = []string{"__EnumValue"}
	__FieldSatisfies           = []string{"__Field"}
	__InputValueSatisfies      = []string{"__InputValue"}
	__SchemaSatisfies          = []string{"__Schema"}
	__TypeSatisfies            = []string{"__Type"}
)

func _droid(ec *executionContext, sel []query.Selection, it *starwars.Droid) {
	groupedFieldSet := ec.collectFields(sel, droidSatisfies, map[string]bool{})
	ec.json.BeginObject()
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "id":
			ec.json.ObjectKey(field.Alias)
			res := it.ID
			ec.json.String(res)
			continue

		case "name":
			ec.json.ObjectKey(field.Alias)
			res := it.Name
			ec.json.String(res)
			continue

		case "friends":
			ec.json.ObjectKey(field.Alias)
			res, err := ec.resolvers.Droid_friends(
				ec.ctx,
				it,
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			ec.json.BeginArray()
			for _, val := range res {
				switch it := val.(type) {
				case nil:
					ec.json.Null()
				case starwars.Human:
					_human(ec, field.Selections, &it)
				case *starwars.Human:
					_human(ec, field.Selections, it)
				case starwars.Droid:
					_droid(ec, field.Selections, &it)
				case *starwars.Droid:
					_droid(ec, field.Selections, it)
				default:
					panic(fmt.Errorf("unexpected type %T", it))
				}
			}
			ec.json.EndArray()
			continue

		case "friendsConnection":
			ec.json.ObjectKey(field.Alias)
			res, err := ec.resolvers.Droid_friendsConnection(
				ec.ctx,
				it,
				field.Args["first"].(*int),
				field.Args["after"].(*string),
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			_friendsConnection(ec, field.Selections, &res)
			continue

		case "appearsIn":
			ec.json.ObjectKey(field.Alias)
			res := it.AppearsIn
			ec.json.BeginArray()
			for _, val := range res {
				ec.json.String(val)
			}
			ec.json.EndArray()
			continue

		case "primaryFunction":
			ec.json.ObjectKey(field.Alias)
			res := it.PrimaryFunction
			ec.json.String(res)
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	ec.json.EndObject()
}

func _friendsConnection(ec *executionContext, sel []query.Selection, it *starwars.FriendsConnection) {
	groupedFieldSet := ec.collectFields(sel, friendsConnectionSatisfies, map[string]bool{})
	ec.json.BeginObject()
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "totalCount":
			ec.json.ObjectKey(field.Alias)
			res := it.TotalCount
			ec.json.Int(res)
			continue

		case "edges":
			ec.json.ObjectKey(field.Alias)
			res := it.Edges
			ec.json.BeginArray()
			for _, val := range res {
				_friendsEdge(ec, field.Selections, &val)
			}
			ec.json.EndArray()
			continue

		case "friends":
			ec.json.ObjectKey(field.Alias)
			res := it.Friends
			ec.json.BeginArray()
			for _, val := range res {
				switch it := val.(type) {
				case nil:
					ec.json.Null()
				case starwars.Human:
					_human(ec, field.Selections, &it)
				case *starwars.Human:
					_human(ec, field.Selections, it)
				case starwars.Droid:
					_droid(ec, field.Selections, &it)
				case *starwars.Droid:
					_droid(ec, field.Selections, it)
				default:
					panic(fmt.Errorf("unexpected type %T", it))
				}
			}
			ec.json.EndArray()
			continue

		case "pageInfo":
			ec.json.ObjectKey(field.Alias)
			res := it.PageInfo
			_pageInfo(ec, field.Selections, &res)
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	ec.json.EndObject()
}

func _friendsEdge(ec *executionContext, sel []query.Selection, it *starwars.FriendsEdge) {
	groupedFieldSet := ec.collectFields(sel, friendsEdgeSatisfies, map[string]bool{})
	ec.json.BeginObject()
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "cursor":
			ec.json.ObjectKey(field.Alias)
			res := it.Cursor
			ec.json.String(res)
			continue

		case "node":
			ec.json.ObjectKey(field.Alias)
			res := it.Node
			switch it := res.(type) {
			case nil:
				ec.json.Null()
			case starwars.Human:
				_human(ec, field.Selections, &it)
			case *starwars.Human:
				_human(ec, field.Selections, it)
			case starwars.Droid:
				_droid(ec, field.Selections, &it)
			case *starwars.Droid:
				_droid(ec, field.Selections, it)
			default:
				panic(fmt.Errorf("unexpected type %T", it))
			}
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	ec.json.EndObject()
}

func _human(ec *executionContext, sel []query.Selection, it *starwars.Human) {
	groupedFieldSet := ec.collectFields(sel, humanSatisfies, map[string]bool{})
	ec.json.BeginObject()
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "id":
			ec.json.ObjectKey(field.Alias)
			res := it.ID
			ec.json.String(res)
			continue

		case "name":
			ec.json.ObjectKey(field.Alias)
			res := it.Name
			ec.json.String(res)
			continue

		case "height":
			ec.json.ObjectKey(field.Alias)
			res := it.Height
			ec.json.Float64(res)
			continue

		case "mass":
			ec.json.ObjectKey(field.Alias)
			res := it.Mass
			ec.json.Float64(res)
			continue

		case "friends":
			ec.json.ObjectKey(field.Alias)
			res, err := ec.resolvers.Human_friends(
				ec.ctx,
				it,
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			ec.json.BeginArray()
			for _, val := range res {
				switch it := val.(type) {
				case nil:
					ec.json.Null()
				case starwars.Human:
					_human(ec, field.Selections, &it)
				case *starwars.Human:
					_human(ec, field.Selections, it)
				case starwars.Droid:
					_droid(ec, field.Selections, &it)
				case *starwars.Droid:
					_droid(ec, field.Selections, it)
				default:
					panic(fmt.Errorf("unexpected type %T", it))
				}
			}
			ec.json.EndArray()
			continue

		case "friendsConnection":
			ec.json.ObjectKey(field.Alias)
			res, err := ec.resolvers.Human_friendsConnection(
				ec.ctx,
				it,
				field.Args["first"].(*int),
				field.Args["after"].(*string),
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			_friendsConnection(ec, field.Selections, &res)
			continue

		case "appearsIn":
			ec.json.ObjectKey(field.Alias)
			res := it.AppearsIn
			ec.json.BeginArray()
			for _, val := range res {
				ec.json.String(val)
			}
			ec.json.EndArray()
			continue

		case "starships":
			ec.json.ObjectKey(field.Alias)
			res, err := ec.resolvers.Human_starships(
				ec.ctx,
				it,
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			ec.json.BeginArray()
			for _, val := range res {
				_starship(ec, field.Selections, &val)
			}
			ec.json.EndArray()
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	ec.json.EndObject()
}

func _mutation(ec *executionContext, sel []query.Selection, it *interface{}) {
	groupedFieldSet := ec.collectFields(sel, mutationSatisfies, map[string]bool{})
	ec.json.BeginObject()
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "createReview":
			ec.json.ObjectKey(field.Alias)
			res, err := ec.resolvers.Mutation_createReview(
				ec.ctx,
				field.Args["episode"].(string),
				field.Args["review"].(starwars.Review),
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			if res == nil {
				ec.json.Null()
			} else {
				_review(ec, field.Selections, res)
			}
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	ec.json.EndObject()
}

func _pageInfo(ec *executionContext, sel []query.Selection, it *starwars.PageInfo) {
	groupedFieldSet := ec.collectFields(sel, pageInfoSatisfies, map[string]bool{})
	ec.json.BeginObject()
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "startCursor":
			ec.json.ObjectKey(field.Alias)
			res := it.StartCursor
			ec.json.String(res)
			continue

		case "endCursor":
			ec.json.ObjectKey(field.Alias)
			res := it.EndCursor
			ec.json.String(res)
			continue

		case "hasNextPage":
			ec.json.ObjectKey(field.Alias)
			res := it.HasNextPage
			ec.json.Bool(res)
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	ec.json.EndObject()
}

func _query(ec *executionContext, sel []query.Selection, it *interface{}) {
	groupedFieldSet := ec.collectFields(sel, querySatisfies, map[string]bool{})
	ec.json.BeginObject()
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "hero":
			ec.json.ObjectKey(field.Alias)
			res, err := ec.resolvers.Query_hero(
				ec.ctx,
				field.Args["episode"].(*string),
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			switch it := res.(type) {
			case nil:
				ec.json.Null()
			case starwars.Human:
				_human(ec, field.Selections, &it)
			case *starwars.Human:
				_human(ec, field.Selections, it)
			case starwars.Droid:
				_droid(ec, field.Selections, &it)
			case *starwars.Droid:
				_droid(ec, field.Selections, it)
			default:
				panic(fmt.Errorf("unexpected type %T", it))
			}
			continue

		case "reviews":
			ec.json.ObjectKey(field.Alias)
			res, err := ec.resolvers.Query_reviews(
				ec.ctx,
				field.Args["episode"].(string),
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			ec.json.BeginArray()
			for _, val := range res {
				_review(ec, field.Selections, &val)
			}
			ec.json.EndArray()
			continue

		case "search":
			ec.json.ObjectKey(field.Alias)
			res, err := ec.resolvers.Query_search(
				ec.ctx,
				field.Args["text"].(string),
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			ec.json.BeginArray()
			for _, val := range res {
				switch it := val.(type) {
				case nil:
					ec.json.Null()
				case starwars.Human:
					_human(ec, field.Selections, &it)
				case *starwars.Human:
					_human(ec, field.Selections, it)
				case starwars.Droid:
					_droid(ec, field.Selections, &it)
				case *starwars.Droid:
					_droid(ec, field.Selections, it)
				case starwars.Starship:
					_starship(ec, field.Selections, &it)
				case *starwars.Starship:
					_starship(ec, field.Selections, it)
				default:
					panic(fmt.Errorf("unexpected type %T", it))
				}
			}
			ec.json.EndArray()
			continue

		case "character":
			ec.json.ObjectKey(field.Alias)
			res, err := ec.resolvers.Query_character(
				ec.ctx,
				field.Args["id"].(string),
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			switch it := res.(type) {
			case nil:
				ec.json.Null()
			case starwars.Human:
				_human(ec, field.Selections, &it)
			case *starwars.Human:
				_human(ec, field.Selections, it)
			case starwars.Droid:
				_droid(ec, field.Selections, &it)
			case *starwars.Droid:
				_droid(ec, field.Selections, it)
			default:
				panic(fmt.Errorf("unexpected type %T", it))
			}
			continue

		case "droid":
			ec.json.ObjectKey(field.Alias)
			res, err := ec.resolvers.Query_droid(
				ec.ctx,
				field.Args["id"].(string),
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			if res == nil {
				ec.json.Null()
			} else {
				_droid(ec, field.Selections, res)
			}
			continue

		case "human":
			ec.json.ObjectKey(field.Alias)
			res, err := ec.resolvers.Query_human(
				ec.ctx,
				field.Args["id"].(string),
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			if res == nil {
				ec.json.Null()
			} else {
				_human(ec, field.Selections, res)
			}
			continue

		case "starship":
			ec.json.ObjectKey(field.Alias)
			res, err := ec.resolvers.Query_starship(
				ec.ctx,
				field.Args["id"].(string),
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			if res == nil {
				ec.json.Null()
			} else {
				_starship(ec, field.Selections, res)
			}
			continue

		case "__schema":
			ec.json.ObjectKey(field.Alias)
			res := ec.introspectSchema()
			if res == nil {
				ec.json.Null()
			} else {
				___Schema(ec, field.Selections, res)
			}
			continue

		case "__type":
			ec.json.ObjectKey(field.Alias)
			res := ec.introspectType(
				field.Args["name"].(string),
			)
			if res == nil {
				ec.json.Null()
			} else {
				___Type(ec, field.Selections, res)
			}
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	ec.json.EndObject()
}

func _review(ec *executionContext, sel []query.Selection, it *starwars.Review) {
	groupedFieldSet := ec.collectFields(sel, reviewSatisfies, map[string]bool{})
	ec.json.BeginObject()
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "stars":
			ec.json.ObjectKey(field.Alias)
			res := it.Stars
			ec.json.Int(res)
			continue

		case "commentary":
			ec.json.ObjectKey(field.Alias)
			res := it.Commentary
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.String(*res)
			}
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	ec.json.EndObject()
}

func _starship(ec *executionContext, sel []query.Selection, it *starwars.Starship) {
	groupedFieldSet := ec.collectFields(sel, starshipSatisfies, map[string]bool{})
	ec.json.BeginObject()
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "id":
			ec.json.ObjectKey(field.Alias)
			res := it.ID
			ec.json.String(res)
			continue

		case "name":
			ec.json.ObjectKey(field.Alias)
			res := it.Name
			ec.json.String(res)
			continue

		case "length":
			ec.json.ObjectKey(field.Alias)
			res := it.Length
			ec.json.Float64(res)
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	ec.json.EndObject()
}

func ___Directive(ec *executionContext, sel []query.Selection, it *introspection.Directive) {
	groupedFieldSet := ec.collectFields(sel, __DirectiveSatisfies, map[string]bool{})
	ec.json.BeginObject()
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "name":
			ec.json.ObjectKey(field.Alias)
			res := it.Name()
			ec.json.String(res)
			continue

		case "description":
			ec.json.ObjectKey(field.Alias)
			res := it.Description()
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.String(*res)
			}
			continue

		case "locations":
			ec.json.ObjectKey(field.Alias)
			res := it.Locations()
			ec.json.BeginArray()
			for _, val := range res {
				ec.json.String(val)
			}
			ec.json.EndArray()
			continue

		case "args":
			ec.json.ObjectKey(field.Alias)
			res := it.Args()
			ec.json.BeginArray()
			for _, val := range res {
				if val == nil {
					ec.json.Null()
				} else {
					___InputValue(ec, field.Selections, val)
				}
			}
			ec.json.EndArray()
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	ec.json.EndObject()
}

func ___EnumValue(ec *executionContext, sel []query.Selection, it *introspection.EnumValue) {
	groupedFieldSet := ec.collectFields(sel, __EnumValueSatisfies, map[string]bool{})
	ec.json.BeginObject()
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "name":
			ec.json.ObjectKey(field.Alias)
			res := it.Name()
			ec.json.String(res)
			continue

		case "description":
			ec.json.ObjectKey(field.Alias)
			res := it.Description()
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.String(*res)
			}
			continue

		case "isDeprecated":
			ec.json.ObjectKey(field.Alias)
			res := it.IsDeprecated()
			ec.json.Bool(res)
			continue

		case "deprecationReason":
			ec.json.ObjectKey(field.Alias)
			res := it.DeprecationReason()
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.String(*res)
			}
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	ec.json.EndObject()
}

func ___Field(ec *executionContext, sel []query.Selection, it *introspection.Field) {
	groupedFieldSet := ec.collectFields(sel, __FieldSatisfies, map[string]bool{})
	ec.json.BeginObject()
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "name":
			ec.json.ObjectKey(field.Alias)
			res := it.Name()
			ec.json.String(res)
			continue

		case "description":
			ec.json.ObjectKey(field.Alias)
			res := it.Description()
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.String(*res)
			}
			continue

		case "args":
			ec.json.ObjectKey(field.Alias)
			res := it.Args()
			ec.json.BeginArray()
			for _, val := range res {
				if val == nil {
					ec.json.Null()
				} else {
					___InputValue(ec, field.Selections, val)
				}
			}
			ec.json.EndArray()
			continue

		case "type":
			ec.json.ObjectKey(field.Alias)
			res := it.Type()
			if res == nil {
				ec.json.Null()
			} else {
				___Type(ec, field.Selections, res)
			}
			continue

		case "isDeprecated":
			ec.json.ObjectKey(field.Alias)
			res := it.IsDeprecated()
			ec.json.Bool(res)
			continue

		case "deprecationReason":
			ec.json.ObjectKey(field.Alias)
			res := it.DeprecationReason()
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.String(*res)
			}
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	ec.json.EndObject()
}

func ___InputValue(ec *executionContext, sel []query.Selection, it *introspection.InputValue) {
	groupedFieldSet := ec.collectFields(sel, __InputValueSatisfies, map[string]bool{})
	ec.json.BeginObject()
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "name":
			ec.json.ObjectKey(field.Alias)
			res := it.Name()
			ec.json.String(res)
			continue

		case "description":
			ec.json.ObjectKey(field.Alias)
			res := it.Description()
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.String(*res)
			}
			continue

		case "type":
			ec.json.ObjectKey(field.Alias)
			res := it.Type()
			if res == nil {
				ec.json.Null()
			} else {
				___Type(ec, field.Selections, res)
			}
			continue

		case "defaultValue":
			ec.json.ObjectKey(field.Alias)
			res := it.DefaultValue()
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.String(*res)
			}
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	ec.json.EndObject()
}

func ___Schema(ec *executionContext, sel []query.Selection, it *introspection.Schema) {
	groupedFieldSet := ec.collectFields(sel, __SchemaSatisfies, map[string]bool{})
	ec.json.BeginObject()
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "types":
			ec.json.ObjectKey(field.Alias)
			res := it.Types()
			ec.json.BeginArray()
			for _, val := range res {
				if val == nil {
					ec.json.Null()
				} else {
					___Type(ec, field.Selections, val)
				}
			}
			ec.json.EndArray()
			continue

		case "queryType":
			ec.json.ObjectKey(field.Alias)
			res := it.QueryType()
			if res == nil {
				ec.json.Null()
			} else {
				___Type(ec, field.Selections, res)
			}
			continue

		case "mutationType":
			ec.json.ObjectKey(field.Alias)
			res := it.MutationType()
			if res == nil {
				ec.json.Null()
			} else {
				___Type(ec, field.Selections, res)
			}
			continue

		case "subscriptionType":
			ec.json.ObjectKey(field.Alias)
			res := it.SubscriptionType()
			if res == nil {
				ec.json.Null()
			} else {
				___Type(ec, field.Selections, res)
			}
			continue

		case "directives":
			ec.json.ObjectKey(field.Alias)
			res := it.Directives()
			ec.json.BeginArray()
			for _, val := range res {
				if val == nil {
					ec.json.Null()
				} else {
					___Directive(ec, field.Selections, val)
				}
			}
			ec.json.EndArray()
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	ec.json.EndObject()
}

func ___Type(ec *executionContext, sel []query.Selection, it *introspection.Type) {
	groupedFieldSet := ec.collectFields(sel, __TypeSatisfies, map[string]bool{})
	ec.json.BeginObject()
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "kind":
			ec.json.ObjectKey(field.Alias)
			res := it.Kind()
			ec.json.String(res)
			continue

		case "name":
			ec.json.ObjectKey(field.Alias)
			res := it.Name()
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.String(*res)
			}
			continue

		case "description":
			ec.json.ObjectKey(field.Alias)
			res := it.Description()
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.String(*res)
			}
			continue

		case "fields":
			ec.json.ObjectKey(field.Alias)
			res := it.Fields(
				field.Args["includeDeprecated"].(bool),
			)
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.BeginArray()
				for _, val := range *res {
					if val == nil {
						ec.json.Null()
					} else {
						___Field(ec, field.Selections, val)
					}
				}
				ec.json.EndArray()
			}
			continue

		case "interfaces":
			ec.json.ObjectKey(field.Alias)
			res := it.Interfaces()
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.BeginArray()
				for _, val := range *res {
					if val == nil {
						ec.json.Null()
					} else {
						___Type(ec, field.Selections, val)
					}
				}
				ec.json.EndArray()
			}
			continue

		case "possibleTypes":
			ec.json.ObjectKey(field.Alias)
			res := it.PossibleTypes()
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.BeginArray()
				for _, val := range *res {
					if val == nil {
						ec.json.Null()
					} else {
						___Type(ec, field.Selections, val)
					}
				}
				ec.json.EndArray()
			}
			continue

		case "enumValues":
			ec.json.ObjectKey(field.Alias)
			res := it.EnumValues(
				field.Args["includeDeprecated"].(bool),
			)
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.BeginArray()
				for _, val := range *res {
					if val == nil {
						ec.json.Null()
					} else {
						___EnumValue(ec, field.Selections, val)
					}
				}
				ec.json.EndArray()
			}
			continue

		case "inputFields":
			ec.json.ObjectKey(field.Alias)
			res := it.InputFields()
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.BeginArray()
				for _, val := range *res {
					if val == nil {
						ec.json.Null()
					} else {
						___InputValue(ec, field.Selections, val)
					}
				}
				ec.json.EndArray()
			}
			continue

		case "ofType":
			ec.json.ObjectKey(field.Alias)
			res := it.OfType()
			if res == nil {
				ec.json.Null()
			} else {
				___Type(ec, field.Selections, res)
			}
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	ec.json.EndObject()
}

var parsedSchema = schema.MustParse("schema {\n    query: Query\n    mutation: Mutation\n}\n# The query type, represents all of the entry points into our object graph\ntype Query {\n    hero(episode: Episode = NEWHOPE): Character\n    reviews(episode: Episode!): [Review]!\n    search(text: String!): [SearchResult]!\n    character(id: ID!): Character\n    droid(id: ID!): Droid\n    human(id: ID!): Human\n    starship(id: ID!): Starship\n}\n# The mutation type, represents all updates we can make to our data\ntype Mutation {\n    createReview(episode: Episode!, review: ReviewInput!): Review\n}\n# The episodes in the Star Wars trilogy\nenum Episode {\n    # Star Wars Episode IV: A New Hope, released in 1977.\n    NEWHOPE\n    # Star Wars Episode V: The Empire Strikes Back, released in 1980.\n    EMPIRE\n    # Star Wars Episode VI: Return of the Jedi, released in 1983.\n    JEDI\n}\n# A character from the Star Wars universe\ninterface Character {\n    # The ID of the character\n    id: ID!\n    # The name of the character\n    name: String!\n    # The friends of the character, or an empty list if they have none\n    friends: [Character]\n    # The friends of the character exposed as a connection with edges\n    friendsConnection(first: Int, after: ID): FriendsConnection!\n    # The movies this character appears in\n    appearsIn: [Episode!]!\n}\n# Units of height\nenum LengthUnit {\n    # The standard unit around the world\n    METER\n    # Primarily used in the United States\n    FOOT\n}\n# A humanoid creature from the Star Wars universe\ntype Human implements Character {\n    # The ID of the human\n    id: ID!\n    # What this human calls themselves\n    name: String!\n    # Height in the preferred unit, default is meters\n    height(unit: LengthUnit = METER): Float!\n    # Mass in kilograms, or null if unknown\n    mass: Float\n    # This human's friends, or an empty list if they have none\n    friends: [Character]\n    # The friends of the human exposed as a connection with edges\n    friendsConnection(first: Int, after: ID): FriendsConnection!\n    # The movies this human appears in\n    appearsIn: [Episode!]!\n    # A list of starships this person has piloted, or an empty list if none\n    starships: [Starship]\n}\n# An autonomous mechanical character in the Star Wars universe\ntype Droid implements Character {\n    # The ID of the droid\n    id: ID!\n    # What others call this droid\n    name: String!\n    # This droid's friends, or an empty list if they have none\n    friends: [Character]\n    # The friends of the droid exposed as a connection with edges\n    friendsConnection(first: Int, after: ID): FriendsConnection!\n    # The movies this droid appears in\n    appearsIn: [Episode!]!\n    # This droid's primary function\n    primaryFunction: String\n}\n# A connection object for a character's friends\ntype FriendsConnection {\n    # The total number of friends\n    totalCount: Int!\n    # The edges for each of the character's friends.\n    edges: [FriendsEdge]\n    # A list of the friends, as a convenience when edges are not needed.\n    friends: [Character]\n    # Information for paginating this connection\n    pageInfo: PageInfo!\n}\n# An edge object for a character's friends\ntype FriendsEdge {\n    # A cursor used for pagination\n    cursor: ID!\n    # The character represented by this friendship edge\n    node: Character\n}\n# Information for paginating this connection\ntype PageInfo {\n    startCursor: ID\n    endCursor: ID\n    hasNextPage: Boolean!\n}\n# Represents a review for a movie\ntype Review {\n    # The number of stars this review gave, 1-5\n    stars: Int!\n    # Comment about the movie\n    commentary: String\n}\n# The input object sent when someone is creating a new review\ninput ReviewInput {\n    # 0-5 stars\n    stars: Int!\n    # Comment about the movie, optional\n    commentary: String\n}\ntype Starship {\n    # The ID of the starship\n    id: ID!\n    # The name of the starship\n    name: String!\n    # Length of the starship, along the longest axis\n    length(unit: LengthUnit = METER): Float!\n}\nunion SearchResult = Human | Droid | Starship\n")
var _ = fmt.Print
