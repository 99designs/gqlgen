//go:generate gorunpkg github.com/vektah/gqlgen -typemap types.json -out generated.go

package chat

import (
	context "context"
	"math/rand"
	"sync"
	"time"
)

type resolvers struct {
	Rooms map[string]*Chatroom
	mu    sync.Mutex
}

func New() *resolvers {
	return &resolvers{
		Rooms: map[string]*Chatroom{},
	}
}

type Chatroom struct {
	Name      string
	Messages  []Message
	Observers map[string]chan Message
}

func (r *resolvers) Mutation_post(ctx context.Context, text string, userName string, roomName string) (Message, error) {
	r.mu.Lock()
	room := r.Rooms[roomName]
	if room == nil {
		room = &Chatroom{Name: roomName, Observers: map[string]chan Message{}}
		r.Rooms[roomName] = room
	}
	r.mu.Unlock()

	message := Message{
		ID:        randString(8),
		CreatedAt: time.Now(),
		Text:      text,
		CreatedBy: userName,
	}

	room.Messages = append(room.Messages, message)
	r.mu.Lock()
	for _, observer := range room.Observers {
		observer <- message
	}
	r.mu.Unlock()
	return message, nil
}

func (r *resolvers) Query_room(ctx context.Context, name string) (*Chatroom, error) {
	r.mu.Lock()
	room := r.Rooms[name]
	if room == nil {
		room = &Chatroom{Name: name, Observers: map[string]chan Message{}}
		r.Rooms[name] = room
	}
	r.mu.Unlock()

	return room, nil
}

func (r *resolvers) Subscription_messageAdded(ctx context.Context, roomName string) (<-chan Message, error) {
	r.mu.Lock()
	room := r.Rooms[roomName]
	if room == nil {
		room = &Chatroom{Name: roomName, Observers: map[string]chan Message{}}
		r.Rooms[roomName] = room
	}
	r.mu.Unlock()

	id := randString(8)
	events := make(chan Message, 1)

	go func() {
		<-ctx.Done()
		r.mu.Lock()
		delete(room.Observers, id)
		r.mu.Unlock()
	}()

	r.mu.Lock()
	room.Observers[id] = events
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
