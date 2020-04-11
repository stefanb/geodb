package services

import (
	"context"
	api "github.com/autom8ter/geodb/gen/go/geodb"
	"github.com/autom8ter/geodb/stream"
	"github.com/dgraph-io/badger/v2"
	"github.com/gogo/protobuf/proto"
	log "github.com/sirupsen/logrus"
)

type GeoDB struct {
	hub *stream.Hub
	db  *badger.DB
}

func NewGeoDB(db *badger.DB, hub *stream.Hub) *GeoDB {
	return &GeoDB{
		hub: hub,
		db:  db,
	}
}

func (p *GeoDB) Ping(ctx context.Context, req *api.PingRequest) (*api.PingResponse, error) {
	return &api.PingResponse{
		Ok: true,
	}, nil
}

func (p *GeoDB) UpsertObjects(ctx context.Context, r *api.UpsertObjectsRequest) (*api.UpsertObjectsResponse, error) {
	txn := p.db.NewTransaction(true)
	defer txn.Discard()
	for k, val := range r.Data {
		bits, _ := proto.Marshal(val)
		if err := txn.Set([]byte(k), bits); err != nil {
			return nil, err
		}
	}
	if err := txn.Commit(); err != nil {
		return nil, err
	}
	return &api.UpsertObjectsResponse{}, nil
}

func (p *GeoDB) GetObjects(ctx context.Context, r *api.GetObjectsRequest) (*api.GetObjectsResponse, error) {
	txn := p.db.NewTransaction(false)
	defer txn.Discard()
	objects := map[string]*api.Object{}
	for _, key := range r.Keys {
		i, err := txn.Get([]byte(key))
		if err != nil {
			return nil, err
		}
		res, err := i.ValueCopy(nil)
		if err != nil {
			return nil, err
		}
		var obj = &api.Object{}
		if err := proto.Unmarshal(res, obj); err != nil {
			return nil, err
		}
		objects[obj.Key] = obj
	}
	return &api.GetObjectsResponse{
		Objects: objects,
	}, nil
}

func (p *GeoDB) DeleteObjects(ctx context.Context, r *api.DeleteObjectsRequest) (*api.DeleteObjectsResponse, error) {
	txn := p.db.NewTransaction(false)
	defer txn.Discard()
	for _, key := range r.Keys {
		if err := txn.Delete([]byte(key)); err != nil {
			return nil, err
		}
	}
	return &api.DeleteObjectsResponse{}, nil
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
