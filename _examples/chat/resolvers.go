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
	Rooms map[string]*Chatroom
	mu    sync.Mutex // nolint: structcheck
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
			Rooms: map[string]*Chatroom{},
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

type Chatroom struct {
	Name      string
	Messages  []Message
	Observers map[string]struct {
		Username string
		Message  chan *Message
	}
}

type mutationResolver struct{ *resolver }

func (r *mutationResolver) Post(ctx context.Context, text string, username string, roomName string) (*Message, error) {
	r.mu.Lock()
	room := r.Rooms[roomName]
	if room == nil {
		room = &Chatroom{
			Name: roomName,
			Observers: map[string]struct {
				Username string
				Message  chan *Message
			}{},
		}
		r.Rooms[roomName] = room
	}
	r.mu.Unlock()

	message := Message{
		ID:        randString(8),
		CreatedAt: time.Now(),
		Text:      text,
		CreatedBy: username,
	}

	room.Messages = append(room.Messages, message)
	r.mu.Lock()
	for _, observer := range room.Observers {
		if observer.Username == "" || observer.Username == message.CreatedBy {
			observer.Message <- &message
		}
	}
	r.mu.Unlock()
	return &message, nil
}

type queryResolver struct{ *resolver }

func (r *queryResolver) Room(ctx context.Context, name string) (*Chatroom, error) {
	r.mu.Lock()
	room := r.Rooms[name]
	if room == nil {
		room = &Chatroom{
			Name: name,
			Observers: map[string]struct {
				Username string
				Message  chan *Message
			}{},
		}
		r.Rooms[name] = room
	}
	r.mu.Unlock()

	return room, nil
}

type subscriptionResolver struct{ *resolver }

func (r *subscriptionResolver) MessageAdded(ctx context.Context, roomName string) (<-chan *Message, error) {
	r.mu.Lock()
	room := r.Rooms[roomName]
	if room == nil {
		room = &Chatroom{
			Name: roomName,
			Observers: map[string]struct {
				Username string
				Message  chan *Message
			}{},
		}
		r.Rooms[roomName] = room
	}
	r.mu.Unlock()

	id := randString(8)
	events := make(chan *Message, 1)

	go func() {
		<-ctx.Done()
		r.mu.Lock()
		delete(room.Observers, id)
		r.mu.Unlock()
	}()

	r.mu.Lock()
	room.Observers[id] = struct {
		Username string
		Message  chan *Message
	}{Username: getUsername(ctx), Message: events}
	r.mu.Unlock()

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
