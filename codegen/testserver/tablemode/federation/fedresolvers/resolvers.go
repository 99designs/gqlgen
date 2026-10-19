// Package fedresolvers implements the federation schema of the tablemode test server once
// for both generated packages, whose resolver interfaces have the same methods.
package fedresolvers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/99designs/gqlgen/codegen/testserver/tablemode/federation/model"
	"github.com/99designs/gqlgen/codegen/testserver/tablemode/resolvers"
	"github.com/99designs/gqlgen/graphql"
)

type Query struct{}

func (Query) Ping(ctx context.Context) (*string, error) { return new("pong"), nil }

type Entity struct{}

func (Entity) FindHelloByName(ctx context.Context, name string) (*model.Hello, error) {
	resolvers.RecordingOf(ctx).Add("entity Hello name=%q", name)
	switch name {
	case "fail":
		return nil, errors.New("no hello called fail")
	case "nil":
		return nil, nil
	}
	return &model.Hello{Name: name, Greeting: "hello " + name}, nil
}

func (Entity) FindWorldByHelloNameAndFoo(
	ctx context.Context,
	helloName string,
	foo string,
) (*model.World, error) {
	resolvers.RecordingOf(ctx).Add("entity World hello.name=%q foo=%q", helloName, foo)
	return &model.World{Foo: foo, Bar: len(foo), Hello: &model.Hello{Name: helloName}}, nil
}

func (Entity) FindWorldByBar(ctx context.Context, bar int) (*model.World, error) {
	resolvers.RecordingOf(ctx).Add("entity World bar=%d", bar)
	return &model.World{Foo: "foo" + strconv.Itoa(bar), Bar: bar}, nil
}

func (Entity) FindPlanetByName(ctx context.Context, name string) (*model.Planet, error) {
	resolvers.RecordingOf(ctx).Add("entity Planet name=%q", name)
	return &model.Planet{Name: name}, nil
}

// FindManyMultiHelloByNames fails for the representations of the name "fail", each with
// an error of its own.
func (Entity) FindManyMultiHelloByNames(
	ctx context.Context,
	reps []*model.MultiHelloByNamesInput,
) ([]*model.MultiHello, error) {
	names := make([]string, len(reps))
	res := make([]*model.MultiHello, len(reps))
	errs := make(graphql.BatchErrorList, len(reps))
	failed := false
	for i, r := range reps {
		names[i] = r.Name
		if r.Name == "fail" {
			errs[i] = fmt.Errorf("no multi hello %d", i)
			failed = true
			continue
		}
		res[i] = &model.MultiHello{Name: r.Name}
	}
	resolvers.RecordingOf(ctx).Add("entity MultiHello names=%s", strings.Join(names, ","))
	if failed {
		return res, errs
	}
	return res, nil
}

func (Entity) FindManyProductByIDs(
	ctx context.Context,
	reps []*model.ProductByIDsInput,
) ([]*model.Product, error) {
	res := make([]*model.Product, len(reps))
	for i, r := range reps {
		resolvers.RecordingOf(ctx).Add("entity Product id=%q category=%q", r.ID, r.Category)
		res[i] = &model.Product{ID: r.ID, Category: r.Category, Display: r.ID + " in " + r.Category}
	}
	return res, nil
}

func (Entity) FindNodeByID(ctx context.Context, id string) (model.Node, error) {
	resolvers.RecordingOf(ctx).Add("entity Node id=%q", id)
	return &model.Item{ID: id, Label: new("node " + id)}, nil
}

func (Entity) FindItemByID(ctx context.Context, id string) (*model.Item, error) {
	resolvers.RecordingOf(ctx).Add("entity Item id=%q", id)
	if id == "none" {
		return nil, errors.New("no item none")
	}
	return &model.Item{ID: id}, nil
}

type Product struct{}

// Tax is a tenth of the price that the representation gives.
func (Product) Tax(
	ctx context.Context,
	obj *model.Product,
	federationRequires map[string]any,
) (*int, error) {
	resolvers.RecordingOf(ctx).Add(
		"field Product.tax id=%q requires=%s",
		obj.ID,
		resolvers.Describe(federationRequires),
	)
	return price(federationRequires, 10)
}

// Discount is a fifth of the price that each representation gives.
func (Product) Discount(
	ctx context.Context,
	objs []*model.Product,
	federationRequires []map[string]any,
) ([]*int, error) {
	ids := make([]string, len(objs))
	for i, o := range objs {
		ids[i] = o.ID
	}
	resolvers.RecordingOf(ctx).Add(
		"batch Product.discount ids=%s requires=%s",
		strings.Join(ids, ","),
		resolvers.Describe(federationRequires),
	)
	res := make([]*int, len(objs))
	for i := range objs {
		if i >= len(federationRequires) {
			return nil, fmt.Errorf("no requires for product %d", i)
		}
		d, err := price(federationRequires[i], 5)
		if err != nil {
			return nil, err
		}
		res[i] = d
	}
	return res, nil
}

// price returns the price of requires divided by div, or nil without a price.
func price(requires map[string]any, div int) (*int, error) {
	switch p := requires["price"].(type) {
	case nil:
		return nil, nil
	case int64:
		return new(int(p) / div), nil
	case float64:
		return new(int(p) / div), nil
	default:
		n, err := strconv.Atoi(fmt.Sprint(p))
		if err != nil {
			return nil, fmt.Errorf("price %v is not a number", p)
		}
		return new(n / div), nil
	}
}
