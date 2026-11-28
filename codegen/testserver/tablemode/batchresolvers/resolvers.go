// Package batchresolvers implements the batch schemas of the tablemode test server once for
// both generated packages of each, whose resolver interfaces have the same methods.
//
// Every batch resolver records the parents of each call and puts the size of its batch
// into what it returns, so that a mode that resolves the parents one by one, or in other
// batches, gives another response than the functions mode.
package batchresolvers

import (
	"context"
	"fmt"

	fsmodel "github.com/99designs/gqlgen/codegen/testserver/tablemode/batchfs/model"
	globalmodel "github.com/99designs/gqlgen/codegen/testserver/tablemode/batchglobal/model"
	valuesmodel "github.com/99designs/gqlgen/codegen/testserver/tablemode/batchvalues/model"
	"github.com/99designs/gqlgen/codegen/testserver/tablemode/resolvers"
)

// GlobalQuery returns three users, the second without posts.
type GlobalQuery struct{}

func (GlobalQuery) Users(ctx context.Context) ([]*globalmodel.User, error) {
	return []*globalmodel.User{
		{ID: "u1", Name: "Luke"},
		{ID: "u2", Name: "Leia"},
		{ID: "u3", Name: "Han"},
	}, nil
}

type GlobalUser struct{}

func (GlobalUser) Posts(
	ctx context.Context,
	objs []*globalmodel.User,
) ([][]*globalmodel.Post, error) {
	posts := map[string][]string{"u1": {"p1", "p2"}, "u3": {"p3"}}
	ids := make([]string, len(objs))
	results := make([][]*globalmodel.Post, len(objs))
	for i, obj := range objs {
		ids[i] = obj.ID
		for _, id := range posts[obj.ID] {
			title := batchID("post by "+obj.Name, id, len(objs))
			results[i] = append(results[i], &globalmodel.Post{ID: id, Title: title})
		}
	}
	resolvers.RecordingOf(ctx).Add("batch User.posts %v", ids)
	return results, nil
}

type GlobalPost struct{}

func (GlobalPost) Author(
	ctx context.Context,
	objs []*globalmodel.Post,
) ([]*globalmodel.Author, error) {
	ids := make([]string, len(objs))
	results := make([]*globalmodel.Author, len(objs))
	for i, obj := range objs {
		ids[i] = obj.ID
		results[i] = &globalmodel.Author{ID: batchID("author", obj.ID, len(objs))}
	}
	resolvers.RecordingOf(ctx).Add("batch Post.author %v", ids)
	return results, nil
}

// ValuesQuery returns users whose posts are values, which the batch resolver of
// Post.author takes as pointers.
type ValuesQuery struct{}

func (ValuesQuery) Users(ctx context.Context) ([]valuesmodel.User, error) {
	return []valuesmodel.User{
		{ID: "u1", Posts: []valuesmodel.Post{{ID: "p1"}, {ID: "p2"}, {ID: "p3"}}},
		{ID: "u2", Posts: []valuesmodel.Post{{ID: "p4"}}},
	}, nil
}

type ValuesPost struct{}

func (ValuesPost) Author(
	ctx context.Context,
	objs []*valuesmodel.Post,
) ([]*valuesmodel.Author, error) {
	ids := make([]string, len(objs))
	results := make([]*valuesmodel.Author, len(objs))
	for i, obj := range objs {
		ids[i] = obj.ID
		results[i] = &valuesmodel.Author{ID: batchID("author", obj.ID, len(objs))}
	}
	resolvers.RecordingOf(ctx).Add("batch Post.author %v", ids)
	return results, nil
}

// FSQuery returns dogs and cats in one list of animals, whose implementations the
// follow-schema layout declares in two files.
type FSQuery struct{}

func (FSQuery) Animals(ctx context.Context) ([]fsmodel.Animal, error) {
	return []fsmodel.Animal{
		&fsmodel.Dog{ID: "d1"},
		&fsmodel.Cat{ID: "c1"},
		fsmodel.Dog{ID: "d2"},
		&fsmodel.Cat{ID: "c2"},
		&fsmodel.Dog{ID: "d3"},
	}, nil
}

type FSDog struct{}

func (FSDog) BatchProp(ctx context.Context, objs []*fsmodel.Dog) ([]string, error) {
	ids := make([]string, len(objs))
	results := make([]string, len(objs))
	for i, obj := range objs {
		ids[i] = obj.ID
		results[i] = batchID("dog", obj.ID, len(objs))
	}
	resolvers.RecordingOf(ctx).Add("batch Dog.batchProp %v", ids)
	return results, nil
}

type FSCat struct{}

func (FSCat) BatchProp(ctx context.Context, objs []*fsmodel.Cat) ([]string, error) {
	ids := make([]string, len(objs))
	results := make([]string, len(objs))
	for i, obj := range objs {
		ids[i] = obj.ID
		results[i] = batchID("cat", obj.ID, len(objs))
	}
	resolvers.RecordingOf(ctx).Add("batch Cat.batchProp %v", ids)
	return results, nil
}

// batchID is the value a batch resolver returns for the parent id in a batch of size
// parents.
func batchID(prefix, id string, size int) string {
	return fmt.Sprintf("%s:%s/%d", prefix, id, size)
}
