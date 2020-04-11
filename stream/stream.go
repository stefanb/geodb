package stream

import (
	"context"
	api "github.com/autom8ter/geodb/gen/go/geodb"
	"github.com/gofrs/uuid"
	"sync"
)

var objectChan = make(chan *api.Object)

type Hub struct {
	clients  map[string]chan *api.Object
	mu       *sync.Mutex
}

func NewHub() *Hub {
	return &Hub{clients: map[string]chan *api.Object{}, mu: &sync.Mutex{}}
}

func (h *Hub) Start(ctx context.Context, channel string) error {
	for {
		select {
		case obj := <-objectChan:
			if h.clients == nil {
				h.clients = map[string]chan *api.Object{}
			}
			for _, channel := range h.clients {
				if channel != nil {
					channel <- obj
				}
			}
		case <-ctx.Done():
			break
		}
	}
}

func (h *Hub) AddMessageStreamClient() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients == nil {
		h.clients = map[string]chan *api.Object{}
	}
	id, _ := uuid.NewV4()
	idStr := id.String()
	h.clients[idStr] = make(chan *api.Object)
	return idStr
}

func (h *Hub) RemoveMessageStreamClient(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[id]; ok {
		close(h.clients[id])
		delete(h.clients, id)
	}
}

func (h *Hub) GetClientMessageStream(id string) chan *api.Object {
	if _, ok := h.clients[id]; ok {
		return h.clients[id]
	}
	return nil
}

func (h *Hub) PublishObject(obj *api.Object) {
	objectChan <- obj
}