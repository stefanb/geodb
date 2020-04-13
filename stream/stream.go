package stream

import (
	"context"
	api "github.com/autom8ter/geodb/gen/go/geodb"
	"github.com/gofrs/uuid"
	"sync"
)

var objectChan = make(chan *api.ObjectDetail)
var eventChan = make(chan *api.Events)

type Hub struct {
	objectClients map[string]chan *api.ObjectDetail
	objMu         *sync.Mutex
	eventClients  map[string]chan *api.Events
	eventMu       *sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		objectClients: map[string]chan *api.ObjectDetail{},
		eventClients:  map[string]chan *api.Events{},
		objMu:         &sync.Mutex{},
		eventMu:       &sync.Mutex{},
	}
}

func (h *Hub) StartObjectStream(ctx context.Context) error {
	for {
		select {
		case obj := <-objectChan:
			if h.objectClients == nil {
				h.objectClients = map[string]chan *api.ObjectDetail{}
			}

			for _, channel := range h.objectClients {
				if channel != nil {
					channel <- obj
				}
			}
		case <-ctx.Done():
			break
		}
	}
}

func (h *Hub) StartEventStream(ctx context.Context) error {
	for {
		select {
		case event := <-eventChan:
			if event == nil {
				continue
			}
			if h.eventClients == nil {
				h.eventClients = map[string]chan *api.Events{}
			}

			for _, channel := range h.eventClients {
				if channel != nil {
					channel <- event
				}
			}
		case <-ctx.Done():
			break
		}
	}
}

func (h *Hub) AddObjectStreamClient(clientID string) string {
	h.objMu.Lock()
	defer h.objMu.Unlock()
	if h.objectClients == nil {
		h.objectClients = map[string]chan *api.ObjectDetail{}
	}
	if clientID == "" {
		id, _ := uuid.NewV4()
		clientID = id.String()
	}
	h.objectClients[clientID] = make(chan *api.ObjectDetail)
	return clientID
}

func (h *Hub) RemoveObjectStreamClient(id string) {
	h.objMu.Lock()
	defer h.objMu.Unlock()
	if _, ok := h.objectClients[id]; ok {
		close(h.objectClients[id])
		delete(h.objectClients, id)
	}
}

func (h *Hub) GetClientObjectStream(id string) chan *api.ObjectDetail {
	h.objMu.Lock()
	defer h.objMu.Unlock()
	if _, ok := h.objectClients[id]; ok {
		return h.objectClients[id]
	}
	return nil
}

func PublishObject(obj *api.ObjectDetail) {
	objectChan <- obj
}

func (h *Hub) PublishObject(obj *api.ObjectDetail) {
	PublishObject(obj)
}
