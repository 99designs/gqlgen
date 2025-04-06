package graph

import (
	"slices"
	"sync"

	"github.com/gqlgen/_examples/mini-habr-with-subscriptions/graph/model"
)

type Subscribers struct {
	Subscribers map[int64][]chan *model.Comment
	mu          sync.Mutex
}

func NewSubscribers() *Subscribers {
	return &Subscribers{
		Subscribers: make(map[int64][]chan *model.Comment), // on postID chans
	}
}

func (s *Subscribers) Pub(postID int64, comment *model.Comment) {
	defer s.mu.Unlock()
	s.mu.Lock()

	chArray, ok := s.Subscribers[postID]
	if !ok {
		return
	}
	var activeChannels []chan *model.Comment

	for _, ch := range chArray {
		func() {
			defer func() {
				if r := recover(); r != nil {
					closeChan(ch) // panic may be due to channel overflow
					return
				}
				activeChannels = append(activeChannels, ch)
			}()
			ch <- comment
		}()
	}

	if len(activeChannels) < len(chArray) {
		s.Subscribers[postID] = activeChannels
	}
}

func (s *Subscribers) Sub(postID int64, ch chan *model.Comment) {
	defer s.mu.Unlock()
	s.mu.Lock()
	s.Subscribers[postID] = append(s.Subscribers[postID], ch)
}

func (s *Subscribers) CloseSub(postID int64, ch chan *model.Comment) {
	defer s.mu.Unlock()
	s.mu.Lock()

	if ch == nil {
		return
	}

	if chArray, ok := s.Subscribers[postID]; ok {
		if idx := slices.Index(chArray, ch); idx != -1 {
			closeChan(ch)
			s.Subscribers[postID] = slices.Delete(chArray, idx, idx+1)
		}
	}

	if len(s.Subscribers[postID]) == 0 {
		delete(s.Subscribers, postID)
	}
}

func closeChan(ch chan *model.Comment) {
	defer func() {
		if r := recover(); r != nil {
			return
		}
	}()
	close(ch)
}
