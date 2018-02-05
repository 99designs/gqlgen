package gen

import (
	"context"
	"fmt"
	"strconv"

	"github.com/vektah/graphql-go/example/starwars"
	"github.com/vektah/graphql-go/introspection"
	"github.com/vektah/graphql-go/jsonw"
	"github.com/vektah/graphql-go/query"
	"github.com/vektah/graphql-go/schema"
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

type droidType struct{}

func (droidType) accepts(name string) bool {
	return true
}

func (t droidType) executeSelectionSet(ec *executionContext, sel []query.Selection, it *starwars.Droid) jsonw.Encodable {
	groupedFieldSet := ec.collectFields(sel, t, map[string]bool{})
	resultMap := jsonw.Map{}
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "id":
			res := it.ID
			json := jsonw.String(res)
			resultMap.Set(field.Alias, json)
			continue

		case "name":
			res := it.Name
			json := jsonw.String(res)
			resultMap.Set(field.Alias, json)
			continue

		case "friends":
			res, err := ec.resolvers.Droid_friends(
				ec.ctx,
				it,
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			json := jsonw.Array{}
			for _, val := range res {
				var json1 jsonw.Encodable = jsonw.Null
				switch it := val.(type) {
				case starwars.Human:
					json1 = humanType{}.executeSelectionSet(ec, field.Selections, &it)
				case *starwars.Human:
					json1 = humanType{}.executeSelectionSet(ec, field.Selections, it)
				case starwars.Droid:
					json1 = droidType{}.executeSelectionSet(ec, field.Selections, &it)
				case *starwars.Droid:
					json1 = droidType{}.executeSelectionSet(ec, field.Selections, it)
				default:
					panic(fmt.Errorf("unexpected type %T", it))
				}
				json = append(json, json1)
			}
			resultMap.Set(field.Alias, json)
			continue

		case "friendsConnection":
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
			json := friendsConnectionType{}.executeSelectionSet(ec, field.Selections, &res)
			resultMap.Set(field.Alias, json)
			continue

		case "appearsIn":
			res := it.AppearsIn
			json := jsonw.Array{}
			for _, val := range res {
				json1 := jsonw.String(val)
				json = append(json, json1)
			}
			resultMap.Set(field.Alias, json)
			continue

		case "primaryFunction":
			res := it.PrimaryFunction
			json := jsonw.String(res)
			resultMap.Set(field.Alias, json)
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	return resultMap
}

type friendsConnectionType struct{}

func (friendsConnectionType) accepts(name string) bool {
	return true
}

func (t friendsConnectionType) executeSelectionSet(ec *executionContext, sel []query.Selection, it *starwars.FriendsConnection) jsonw.Encodable {
	groupedFieldSet := ec.collectFields(sel, t, map[string]bool{})
	resultMap := jsonw.Map{}
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "totalCount":
			res := it.TotalCount
			json := jsonw.Int(res)
			resultMap.Set(field.Alias, json)
			continue

		case "edges":
			res := it.Edges
			json := jsonw.Array{}
			for _, val := range res {
				json1 := friendsEdgeType{}.executeSelectionSet(ec, field.Selections, &val)
				json = append(json, json1)
			}
			resultMap.Set(field.Alias, json)
			continue

		case "friends":
			res := it.Friends
			json := jsonw.Array{}
			for _, val := range res {
				var json1 jsonw.Encodable = jsonw.Null
				switch it := val.(type) {
				case starwars.Human:
					json1 = humanType{}.executeSelectionSet(ec, field.Selections, &it)
				case *starwars.Human:
					json1 = humanType{}.executeSelectionSet(ec, field.Selections, it)
				case starwars.Droid:
					json1 = droidType{}.executeSelectionSet(ec, field.Selections, &it)
				case *starwars.Droid:
					json1 = droidType{}.executeSelectionSet(ec, field.Selections, it)
				default:
					panic(fmt.Errorf("unexpected type %T", it))
				}
				json = append(json, json1)
			}
			resultMap.Set(field.Alias, json)
			continue

		case "pageInfo":
			res := it.PageInfo
			json := pageInfoType{}.executeSelectionSet(ec, field.Selections, &res)
			resultMap.Set(field.Alias, json)
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	return resultMap
}

type friendsEdgeType struct{}

func (friendsEdgeType) accepts(name string) bool {
	return true
}

func (t friendsEdgeType) executeSelectionSet(ec *executionContext, sel []query.Selection, it *starwars.FriendsEdge) jsonw.Encodable {
	groupedFieldSet := ec.collectFields(sel, t, map[string]bool{})
	resultMap := jsonw.Map{}
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "cursor":
			res := it.Cursor
			json := jsonw.String(res)
			resultMap.Set(field.Alias, json)
			continue

		case "node":
			res := it.Node
			var json jsonw.Encodable = jsonw.Null
			switch it := res.(type) {
			case starwars.Human:
				json = humanType{}.executeSelectionSet(ec, field.Selections, &it)
			case *starwars.Human:
				json = humanType{}.executeSelectionSet(ec, field.Selections, it)
			case starwars.Droid:
				json = droidType{}.executeSelectionSet(ec, field.Selections, &it)
			case *starwars.Droid:
				json = droidType{}.executeSelectionSet(ec, field.Selections, it)
			default:
				panic(fmt.Errorf("unexpected type %T", it))
			}
			resultMap.Set(field.Alias, json)
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	return resultMap
}

type humanType struct{}

func (humanType) accepts(name string) bool {
	return true
}

func (t humanType) executeSelectionSet(ec *executionContext, sel []query.Selection, it *starwars.Human) jsonw.Encodable {
	groupedFieldSet := ec.collectFields(sel, t, map[string]bool{})
	resultMap := jsonw.Map{}
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "id":
			res := it.ID
			json := jsonw.String(res)
			resultMap.Set(field.Alias, json)
			continue

		case "name":
			res := it.Name
			json := jsonw.String(res)
			resultMap.Set(field.Alias, json)
			continue

		case "height":
			res := it.Height
			json := jsonw.Float64(res)
			resultMap.Set(field.Alias, json)
			continue

		case "mass":
			res := it.Mass
			json := jsonw.Float64(res)
			resultMap.Set(field.Alias, json)
			continue

		case "friends":
			res, err := ec.resolvers.Human_friends(
				ec.ctx,
				it,
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			json := jsonw.Array{}
			for _, val := range res {
				var json1 jsonw.Encodable = jsonw.Null
				switch it := val.(type) {
				case starwars.Human:
					json1 = humanType{}.executeSelectionSet(ec, field.Selections, &it)
				case *starwars.Human:
					json1 = humanType{}.executeSelectionSet(ec, field.Selections, it)
				case starwars.Droid:
					json1 = droidType{}.executeSelectionSet(ec, field.Selections, &it)
				case *starwars.Droid:
					json1 = droidType{}.executeSelectionSet(ec, field.Selections, it)
				default:
					panic(fmt.Errorf("unexpected type %T", it))
				}
				json = append(json, json1)
			}
			resultMap.Set(field.Alias, json)
			continue

		case "friendsConnection":
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
			json := friendsConnectionType{}.executeSelectionSet(ec, field.Selections, &res)
			resultMap.Set(field.Alias, json)
			continue

		case "appearsIn":
			res := it.AppearsIn
			json := jsonw.Array{}
			for _, val := range res {
				json1 := jsonw.String(val)
				json = append(json, json1)
			}
			resultMap.Set(field.Alias, json)
			continue

		case "starships":
			res, err := ec.resolvers.Human_starships(
				ec.ctx,
				it,
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			json := jsonw.Array{}
			for _, val := range res {
				json1 := starshipType{}.executeSelectionSet(ec, field.Selections, &val)
				json = append(json, json1)
			}
			resultMap.Set(field.Alias, json)
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	return resultMap
}

type mutationType struct{}

func (mutationType) accepts(name string) bool {
	return true
}

func (t mutationType) executeSelectionSet(ec *executionContext, sel []query.Selection, it *interface{}) jsonw.Encodable {
	groupedFieldSet := ec.collectFields(sel, t, map[string]bool{})
	resultMap := jsonw.Map{}
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "createReview":
			res, err := ec.resolvers.Mutation_createReview(
				ec.ctx,
				field.Args["episode"].(string),
				field.Args["review"].(starwars.Review),
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := reviewType{}.executeSelectionSet(ec, field.Selections, res)
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	return resultMap
}

type pageInfoType struct{}

func (pageInfoType) accepts(name string) bool {
	return true
}

func (t pageInfoType) executeSelectionSet(ec *executionContext, sel []query.Selection, it *starwars.PageInfo) jsonw.Encodable {
	groupedFieldSet := ec.collectFields(sel, t, map[string]bool{})
	resultMap := jsonw.Map{}
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "startCursor":
			res := it.StartCursor
			json := jsonw.String(res)
			resultMap.Set(field.Alias, json)
			continue

		case "endCursor":
			res := it.EndCursor
			json := jsonw.String(res)
			resultMap.Set(field.Alias, json)
			continue

		case "hasNextPage":
			res := it.HasNextPage
			json := jsonw.Bool(res)
			resultMap.Set(field.Alias, json)
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	return resultMap
}

type queryType struct{}

func (queryType) accepts(name string) bool {
	return true
}

func (t queryType) executeSelectionSet(ec *executionContext, sel []query.Selection, it *interface{}) jsonw.Encodable {
	groupedFieldSet := ec.collectFields(sel, t, map[string]bool{})
	resultMap := jsonw.Map{}
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "hero":
			res, err := ec.resolvers.Query_hero(
				ec.ctx,
				field.Args["episode"].(*string),
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			var json jsonw.Encodable = jsonw.Null
			switch it := res.(type) {
			case starwars.Human:
				json = humanType{}.executeSelectionSet(ec, field.Selections, &it)
			case *starwars.Human:
				json = humanType{}.executeSelectionSet(ec, field.Selections, it)
			case starwars.Droid:
				json = droidType{}.executeSelectionSet(ec, field.Selections, &it)
			case *starwars.Droid:
				json = droidType{}.executeSelectionSet(ec, field.Selections, it)
			default:
				panic(fmt.Errorf("unexpected type %T", it))
			}
			resultMap.Set(field.Alias, json)
			continue

		case "reviews":
			res, err := ec.resolvers.Query_reviews(
				ec.ctx,
				field.Args["episode"].(string),
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			json := jsonw.Array{}
			for _, val := range res {
				json1 := reviewType{}.executeSelectionSet(ec, field.Selections, &val)
				json = append(json, json1)
			}
			resultMap.Set(field.Alias, json)
			continue

		case "search":
			res, err := ec.resolvers.Query_search(
				ec.ctx,
				field.Args["text"].(string),
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			json := jsonw.Array{}
			for _, val := range res {
				var json1 jsonw.Encodable = jsonw.Null
				switch it := val.(type) {
				case starwars.Human:
					json1 = humanType{}.executeSelectionSet(ec, field.Selections, &it)
				case *starwars.Human:
					json1 = humanType{}.executeSelectionSet(ec, field.Selections, it)
				case starwars.Droid:
					json1 = droidType{}.executeSelectionSet(ec, field.Selections, &it)
				case *starwars.Droid:
					json1 = droidType{}.executeSelectionSet(ec, field.Selections, it)
				case starwars.Starship:
					json1 = starshipType{}.executeSelectionSet(ec, field.Selections, &it)
				case *starwars.Starship:
					json1 = starshipType{}.executeSelectionSet(ec, field.Selections, it)
				default:
					panic(fmt.Errorf("unexpected type %T", it))
				}
				json = append(json, json1)
			}
			resultMap.Set(field.Alias, json)
			continue

		case "character":
			res, err := ec.resolvers.Query_character(
				ec.ctx,
				field.Args["id"].(string),
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			var json jsonw.Encodable = jsonw.Null
			switch it := res.(type) {
			case starwars.Human:
				json = humanType{}.executeSelectionSet(ec, field.Selections, &it)
			case *starwars.Human:
				json = humanType{}.executeSelectionSet(ec, field.Selections, it)
			case starwars.Droid:
				json = droidType{}.executeSelectionSet(ec, field.Selections, &it)
			case *starwars.Droid:
				json = droidType{}.executeSelectionSet(ec, field.Selections, it)
			default:
				panic(fmt.Errorf("unexpected type %T", it))
			}
			resultMap.Set(field.Alias, json)
			continue

		case "droid":
			res, err := ec.resolvers.Query_droid(
				ec.ctx,
				field.Args["id"].(string),
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := droidType{}.executeSelectionSet(ec, field.Selections, res)
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		case "human":
			res, err := ec.resolvers.Query_human(
				ec.ctx,
				field.Args["id"].(string),
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := humanType{}.executeSelectionSet(ec, field.Selections, res)
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		case "starship":
			res, err := ec.resolvers.Query_starship(
				ec.ctx,
				field.Args["id"].(string),
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := starshipType{}.executeSelectionSet(ec, field.Selections, res)
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		case "__schema":
			res := ec.introspectSchema()
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := __SchemaType{}.executeSelectionSet(ec, field.Selections, res)
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		case "__type":
			res := ec.introspectType(
				field.Args["name"].(string),
			)
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := __TypeType{}.executeSelectionSet(ec, field.Selections, res)
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	return resultMap
}

type reviewType struct{}

func (reviewType) accepts(name string) bool {
	return true
}

func (t reviewType) executeSelectionSet(ec *executionContext, sel []query.Selection, it *starwars.Review) jsonw.Encodable {
	groupedFieldSet := ec.collectFields(sel, t, map[string]bool{})
	resultMap := jsonw.Map{}
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "stars":
			res := it.Stars
			json := jsonw.Int(res)
			resultMap.Set(field.Alias, json)
			continue

		case "commentary":
			res := it.Commentary
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := jsonw.String(*res)
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	return resultMap
}

type starshipType struct{}

func (starshipType) accepts(name string) bool {
	return true
}

func (t starshipType) executeSelectionSet(ec *executionContext, sel []query.Selection, it *starwars.Starship) jsonw.Encodable {
	groupedFieldSet := ec.collectFields(sel, t, map[string]bool{})
	resultMap := jsonw.Map{}
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "id":
			res := it.ID
			json := jsonw.String(res)
			resultMap.Set(field.Alias, json)
			continue

		case "name":
			res := it.Name
			json := jsonw.String(res)
			resultMap.Set(field.Alias, json)
			continue

		case "length":
			res := it.Length
			json := jsonw.Float64(res)
			resultMap.Set(field.Alias, json)
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	return resultMap
}

type __DirectiveType struct{}

func (__DirectiveType) accepts(name string) bool {
	return true
}

func (t __DirectiveType) executeSelectionSet(ec *executionContext, sel []query.Selection, it *introspection.Directive) jsonw.Encodable {
	groupedFieldSet := ec.collectFields(sel, t, map[string]bool{})
	resultMap := jsonw.Map{}
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "name":
			res := it.Name()
			json := jsonw.String(res)
			resultMap.Set(field.Alias, json)
			continue

		case "description":
			res := it.Description()
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := jsonw.String(*res)
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		case "locations":
			res := it.Locations()
			json := jsonw.Array{}
			for _, val := range res {
				json1 := jsonw.String(val)
				json = append(json, json1)
			}
			resultMap.Set(field.Alias, json)
			continue

		case "args":
			res := it.Args()
			json := jsonw.Array{}
			for _, val := range res {
				var json1 jsonw.Encodable = jsonw.Null
				if val != nil {
					json11 := __InputValueType{}.executeSelectionSet(ec, field.Selections, val)
					json1 = json11
				}
				json = append(json, json1)
			}
			resultMap.Set(field.Alias, json)
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	return resultMap
}

type __EnumValueType struct{}

func (__EnumValueType) accepts(name string) bool {
	return true
}

func (t __EnumValueType) executeSelectionSet(ec *executionContext, sel []query.Selection, it *introspection.EnumValue) jsonw.Encodable {
	groupedFieldSet := ec.collectFields(sel, t, map[string]bool{})
	resultMap := jsonw.Map{}
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "name":
			res := it.Name()
			json := jsonw.String(res)
			resultMap.Set(field.Alias, json)
			continue

		case "description":
			res := it.Description()
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := jsonw.String(*res)
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		case "isDeprecated":
			res := it.IsDeprecated()
			json := jsonw.Bool(res)
			resultMap.Set(field.Alias, json)
			continue

		case "deprecationReason":
			res := it.DeprecationReason()
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := jsonw.String(*res)
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	return resultMap
}

type __FieldType struct{}

func (__FieldType) accepts(name string) bool {
	return true
}

func (t __FieldType) executeSelectionSet(ec *executionContext, sel []query.Selection, it *introspection.Field) jsonw.Encodable {
	groupedFieldSet := ec.collectFields(sel, t, map[string]bool{})
	resultMap := jsonw.Map{}
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "name":
			res := it.Name()
			json := jsonw.String(res)
			resultMap.Set(field.Alias, json)
			continue

		case "description":
			res := it.Description()
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := jsonw.String(*res)
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		case "args":
			res := it.Args()
			json := jsonw.Array{}
			for _, val := range res {
				var json1 jsonw.Encodable = jsonw.Null
				if val != nil {
					json11 := __InputValueType{}.executeSelectionSet(ec, field.Selections, val)
					json1 = json11
				}
				json = append(json, json1)
			}
			resultMap.Set(field.Alias, json)
			continue

		case "type":
			res := it.Type()
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := __TypeType{}.executeSelectionSet(ec, field.Selections, res)
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		case "isDeprecated":
			res := it.IsDeprecated()
			json := jsonw.Bool(res)
			resultMap.Set(field.Alias, json)
			continue

		case "deprecationReason":
			res := it.DeprecationReason()
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := jsonw.String(*res)
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	return resultMap
}

type __InputValueType struct{}

func (__InputValueType) accepts(name string) bool {
	return true
}

func (t __InputValueType) executeSelectionSet(ec *executionContext, sel []query.Selection, it *introspection.InputValue) jsonw.Encodable {
	groupedFieldSet := ec.collectFields(sel, t, map[string]bool{})
	resultMap := jsonw.Map{}
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "name":
			res := it.Name()
			json := jsonw.String(res)
			resultMap.Set(field.Alias, json)
			continue

		case "description":
			res := it.Description()
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := jsonw.String(*res)
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		case "type":
			res := it.Type()
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := __TypeType{}.executeSelectionSet(ec, field.Selections, res)
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		case "defaultValue":
			res := it.DefaultValue()
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := jsonw.String(*res)
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	return resultMap
}

type __SchemaType struct{}

func (__SchemaType) accepts(name string) bool {
	return true
}

func (t __SchemaType) executeSelectionSet(ec *executionContext, sel []query.Selection, it *introspection.Schema) jsonw.Encodable {
	groupedFieldSet := ec.collectFields(sel, t, map[string]bool{})
	resultMap := jsonw.Map{}
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "types":
			res := it.Types()
			json := jsonw.Array{}
			for _, val := range res {
				var json1 jsonw.Encodable = jsonw.Null
				if val != nil {
					json11 := __TypeType{}.executeSelectionSet(ec, field.Selections, val)
					json1 = json11
				}
				json = append(json, json1)
			}
			resultMap.Set(field.Alias, json)
			continue

		case "queryType":
			res := it.QueryType()
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := __TypeType{}.executeSelectionSet(ec, field.Selections, res)
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		case "mutationType":
			res := it.MutationType()
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := __TypeType{}.executeSelectionSet(ec, field.Selections, res)
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		case "subscriptionType":
			res := it.SubscriptionType()
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := __TypeType{}.executeSelectionSet(ec, field.Selections, res)
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		case "directives":
			res := it.Directives()
			json := jsonw.Array{}
			for _, val := range res {
				var json1 jsonw.Encodable = jsonw.Null
				if val != nil {
					json11 := __DirectiveType{}.executeSelectionSet(ec, field.Selections, val)
					json1 = json11
				}
				json = append(json, json1)
			}
			resultMap.Set(field.Alias, json)
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	return resultMap
}

type __TypeType struct{}

func (__TypeType) accepts(name string) bool {
	return true
}

func (t __TypeType) executeSelectionSet(ec *executionContext, sel []query.Selection, it *introspection.Type) jsonw.Encodable {
	groupedFieldSet := ec.collectFields(sel, t, map[string]bool{})
	resultMap := jsonw.Map{}
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "kind":
			res := it.Kind()
			json := jsonw.String(res)
			resultMap.Set(field.Alias, json)
			continue

		case "name":
			res := it.Name()
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := jsonw.String(*res)
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		case "description":
			res := it.Description()
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := jsonw.String(*res)
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		case "fields":
			res := it.Fields(
				field.Args["includeDeprecated"].(bool),
			)
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := jsonw.Array{}
				for _, val := range *res {
					var json11 jsonw.Encodable = jsonw.Null
					if val != nil {
						json111 := __FieldType{}.executeSelectionSet(ec, field.Selections, val)
						json11 = json111
					}
					json1 = append(json1, json11)
				}
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		case "interfaces":
			res := it.Interfaces()
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := jsonw.Array{}
				for _, val := range *res {
					var json11 jsonw.Encodable = jsonw.Null
					if val != nil {
						json111 := __TypeType{}.executeSelectionSet(ec, field.Selections, val)
						json11 = json111
					}
					json1 = append(json1, json11)
				}
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		case "possibleTypes":
			res := it.PossibleTypes()
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := jsonw.Array{}
				for _, val := range *res {
					var json11 jsonw.Encodable = jsonw.Null
					if val != nil {
						json111 := __TypeType{}.executeSelectionSet(ec, field.Selections, val)
						json11 = json111
					}
					json1 = append(json1, json11)
				}
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		case "enumValues":
			res := it.EnumValues(
				field.Args["includeDeprecated"].(bool),
			)
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := jsonw.Array{}
				for _, val := range *res {
					var json11 jsonw.Encodable = jsonw.Null
					if val != nil {
						json111 := __EnumValueType{}.executeSelectionSet(ec, field.Selections, val)
						json11 = json111
					}
					json1 = append(json1, json11)
				}
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		case "inputFields":
			res := it.InputFields()
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := jsonw.Array{}
				for _, val := range *res {
					var json11 jsonw.Encodable = jsonw.Null
					if val != nil {
						json111 := __InputValueType{}.executeSelectionSet(ec, field.Selections, val)
						json11 = json111
					}
					json1 = append(json1, json11)
				}
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		case "ofType":
			res := it.OfType()
			var json jsonw.Encodable = jsonw.Null
			if res != nil {
				json1 := __TypeType{}.executeSelectionSet(ec, field.Selections, res)
				json = json1
			}
			resultMap.Set(field.Alias, json)
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	return resultMap
}

var parsedSchema = schema.MustParse("schema {\n    query: Query\n    mutation: Mutation\n}\n# The query type, represents all of the entry points into our object graph\ntype Query {\n    hero(episode: Episode = NEWHOPE): Character\n    reviews(episode: Episode!): [Review]!\n    search(text: String!): [SearchResult]!\n    character(id: ID!): Character\n    droid(id: ID!): Droid\n    human(id: ID!): Human\n    starship(id: ID!): Starship\n}\n# The mutation type, represents all updates we can make to our data\ntype Mutation {\n    createReview(episode: Episode!, review: ReviewInput!): Review\n}\n# The episodes in the Star Wars trilogy\nenum Episode {\n    # Star Wars Episode IV: A New Hope, released in 1977.\n    NEWHOPE\n    # Star Wars Episode V: The Empire Strikes Back, released in 1980.\n    EMPIRE\n    # Star Wars Episode VI: Return of the Jedi, released in 1983.\n    JEDI\n}\n# A character from the Star Wars universe\ninterface Character {\n    # The ID of the character\n    id: ID!\n    # The name of the character\n    name: String!\n    # The friends of the character, or an empty list if they have none\n    friends: [Character]\n    # The friends of the character exposed as a connection with edges\n    friendsConnection(first: Int, after: ID): FriendsConnection!\n    # The movies this character appears in\n    appearsIn: [Episode!]!\n}\n# Units of height\nenum LengthUnit {\n    # The standard unit around the world\n    METER\n    # Primarily used in the United States\n    FOOT\n}\n# A humanoid creature from the Star Wars universe\ntype Human implements Character {\n    # The ID of the human\n    id: ID!\n    # What this human calls themselves\n    name: String!\n    # Height in the preferred unit, default is meters\n    height(unit: LengthUnit = METER): Float!\n    # Mass in kilograms, or null if unknown\n    mass: Float\n    # This human's friends, or an empty list if they have none\n    friends: [Character]\n    # The friends of the human exposed as a connection with edges\n    friendsConnection(first: Int, after: ID): FriendsConnection!\n    # The movies this human appears in\n    appearsIn: [Episode!]!\n    # A list of starships this person has piloted, or an empty list if none\n    starships: [Starship]\n}\n# An autonomous mechanical character in the Star Wars universe\ntype Droid implements Character {\n    # The ID of the droid\n    id: ID!\n    # What others call this droid\n    name: String!\n    # This droid's friends, or an empty list if they have none\n    friends: [Character]\n    # The friends of the droid exposed as a connection with edges\n    friendsConnection(first: Int, after: ID): FriendsConnection!\n    # The movies this droid appears in\n    appearsIn: [Episode!]!\n    # This droid's primary function\n    primaryFunction: String\n}\n# A connection object for a character's friends\ntype FriendsConnection {\n    # The total number of friends\n    totalCount: Int!\n    # The edges for each of the character's friends.\n    edges: [FriendsEdge]\n    # A list of the friends, as a convenience when edges are not needed.\n    friends: [Character]\n    # Information for paginating this connection\n    pageInfo: PageInfo!\n}\n# An edge object for a character's friends\ntype FriendsEdge {\n    # A cursor used for pagination\n    cursor: ID!\n    # The character represented by this friendship edge\n    node: Character\n}\n# Information for paginating this connection\ntype PageInfo {\n    startCursor: ID\n    endCursor: ID\n    hasNextPage: Boolean!\n}\n# Represents a review for a movie\ntype Review {\n    # The number of stars this review gave, 1-5\n    stars: Int!\n    # Comment about the movie\n    commentary: String\n}\n# The input object sent when someone is creating a new review\ninput ReviewInput {\n    # 0-5 stars\n    stars: Int!\n    # Comment about the movie, optional\n    commentary: String\n}\ntype Starship {\n    # The ID of the starship\n    id: ID!\n    # The name of the starship\n    name: String!\n    # Length of the starship, along the longest axis\n    length(unit: LengthUnit = METER): Float!\n}\nunion SearchResult = Human | Droid | Starship\n")
var _ = fmt.Print
