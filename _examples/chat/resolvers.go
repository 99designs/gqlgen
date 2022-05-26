//go:generate go run ../../testdata/gqlgen.go

package chat

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/99designs/gqlgen/graphql"
)

type ckey string

type resolver struct {
	Rooms sync.Map
}

func (r *resolver) Mutation() MutationResolver {
	return &mutationResolver{r}
}

func (r *resolver) Query() QueryResolver {
	return &queryResolver{r}
}

func (r *resolver) Subscription() SubscriptionResolver {
	return &subscriptionResolver{r}
}

func New() Config {
	return Config{
		Resolvers: &resolver{
			Rooms: sync.Map{},
		},
		Directives: DirectiveRoot{
			User: func(ctx context.Context, obj interface{}, next graphql.Resolver, username string) (res interface{}, err error) {
				return next(context.WithValue(ctx, ckey("username"), username))
			},
		},
	}
}

func getUsername(ctx context.Context) string {
	if username, ok := ctx.Value(ckey("username")).(string); ok {
		return username
	}
	return ""
}

type Observer struct {
	Username string
	Message  chan *Message
}

type Chatroom struct {
	Name      string
	Messages  []Message
	Observers sync.Map
}

type mutationResolver struct{ *resolver }

func (r *mutationResolver) Post(ctx context.Context, text string, username string, roomName string) (*Message, error) {
	room := r.getRoom(roomName)

	message := &Message{
		ID:        randString(8),
		CreatedAt: time.Now(),
		Text:      text,
		CreatedBy: username,
	}

	room.Messages = append(room.Messages, *message)
	room.Observers.Range(func(_, v interface{}) bool {
		observer := v.(*Observer)
		if observer.Username == "" || observer.Username == message.CreatedBy {
			observer.Message <- message
		}
		return true
	})
	return message, nil
}

type queryResolver struct{ *resolver }

func (r *resolver) getRoom(name string) *Chatroom {
	room, _ := r.Rooms.LoadOrStore(name, &Chatroom{
		Name:      name,
		Observers: sync.Map{},
	})
	return room.(*Chatroom)
}

func (r *queryResolver) Room(ctx context.Context, name string) (*Chatroom, error) {
	return r.getRoom(name), nil
}

type subscriptionResolver struct{ *resolver }

func (r *subscriptionResolver) MessageAdded(ctx context.Context, roomName string) (<-chan *Message, error) {
	room := r.getRoom(roomName)

	id := randString(8)
	events := make(chan *Message, 1)

	go func() {
		<-ctx.Done()
		room.Observers.Delete(id)
	}()

	room.Observers.Store(id, &Observer{
		Username: getUsername(ctx),
		Message:  events,
	})

	events <- &Message{
		ID:        randString(8),
		CreatedAt: time.Now(),
		Text:      "You've joined the room",
		CreatedBy: "system",
	}

	return events, nil
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
