//go:generate gorunpkg github.com/vektah/gqlgen -out generated.go

package chat

import (
	context "context"
	"math/rand"
	"time"
)

type resolvers struct {
	Rooms map[string]*Chatroom
}

func New() *resolvers {
	return &resolvers{
		Rooms: map[string]*Chatroom{},
	}
}

func (r *resolvers) Mutation_post(ctx context.Context, text string, userName string, roomName string) (Message, error) {
	room := r.Rooms[roomName]
	if room == nil {
		room = &Chatroom{Name: roomName, Observers: map[string]chan Message{}}
		r.Rooms[roomName] = room
	}

	message := Message{
		ID:        randString(8),
		CreatedAt: time.Now(),
		Text:      text,
		CreatedBy: userName,
	}

	room.Messages = append(room.Messages, message)
	room.mu.Lock()
	for _, observer := range room.Observers {
		observer <- message
	}
	room.mu.Unlock()
	return message, nil
}

func (r *resolvers) Query_room(ctx context.Context, name string) (*Chatroom, error) {
	room := r.Rooms[name]
	if room == nil {
		room = &Chatroom{Name: name, Observers: map[string]chan Message{}}
		r.Rooms[name] = room
	}

	return room, nil
}

func (r *resolvers) Subscription_messageAdded(ctx context.Context, roomName string) (<-chan Message, error) {
	room := r.Rooms[roomName]
	if room == nil {
		room = &Chatroom{Name: roomName, Observers: map[string]chan Message{}}
		r.Rooms[roomName] = room
	}

	id := randString(8)
	events := make(chan Message, 1)

	go func() {
		<-ctx.Done()
		room.mu.Lock()
		delete(room.Observers, id)
		room.mu.Unlock()
	}()

	room.mu.Lock()
	room.Observers[id] = events
	room.mu.Unlock()

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
