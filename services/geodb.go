package services

import (
	"context"
	api "github.com/autom8ter/geodb/gen/go/geodb"
	"github.com/autom8ter/geodb/stream"
	log "github.com/sirupsen/logrus"
	"github.com/dgraph-io/badger/v2"
)

type GeoDB struct {
	hub *stream.Hub
	db *badger.DB
}

func NewGeoDB(db *badger.DB, hub *stream.Hub) *GeoDB {
	return &GeoDB{
		hub: hub,
		db: db,
	}
}

func (p *GeoDB) Ping(ctx context.Context, req *api.PingRequest) (*api.PingResponse, error) {
	return &api.PingResponse{
		Ok: true,
	}, nil
}


func (p *GeoDB) UpsertObjects(context.Context, *api.UpsertObjectsRequest) (*api.UpsertObjectsResponse, error) {
	panic("implement me")
}

func (p *GeoDB) GetObjects(context.Context, *api.GetObjectsRequest) (*api.GetObjectsResponse, error) {
	panic("implement me")
}

func (p *GeoDB) DeleteObjects(context.Context, *api.DeleteObjectsRequest) (*api.DeleteObjectsResponse, error) {
	panic("implement me")
}

func (p *GeoDB) StreamObjects(r *api.StreamObjectsRequest, ss api.GeoDB_StreamObjectsServer) error {
	clientID := p.hub.AddMessageStreamClient()
	for {
		select {
		case msg := <-p.hub.GetClientMessageStream(clientID):
			if err := ss.Send(&api.StreamObjectsResponse{
				Object: msg,
			}); err != nil {
				log.Error(err.Error())
			}
		case <-ss.Context().Done():
			p.hub.RemoveMessageStreamClient(clientID)
			break
		}
	}
}
