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

func (p *GeoDB) Upsert(ctx context.Context, r *api.UpsertRequest) (*api.UpsertResponse, error) {
	txn := p.db.NewTransaction(true)
	defer txn.Discard()
	for k, val := range r.Data {
		val.Key = k
		bits, _ := proto.Marshal(val)
		if err := txn.Set([]byte(k), bits); err != nil {
			return nil, err
		}
		p.hub.PublishObject(val)
	}
	if err := txn.Commit(); err != nil {
		return nil, err
	}
	return &api.UpsertResponse{}, nil
}

func (p *GeoDB) Get(ctx context.Context, r *api.GetRequest) (*api.GetResponse, error) {
	txn := p.db.NewTransaction(false)
	defer txn.Discard()
	objects := map[string]*api.Data{}
	for _, key := range r.Keys {
		i, err := txn.Get([]byte(key))
		if err != nil {
			return nil, err
		}
		res, err := i.ValueCopy(nil)
		if err != nil {
			return nil, err
		}
		var obj = &api.Data{}
		if err := proto.Unmarshal(res, obj); err != nil {
			return nil, err
		}
		objects[key] = obj
	}
	return &api.GetResponse{
		Data: objects,
	}, nil
}

func (p *GeoDB) Delete(ctx context.Context, r *api.DeleteRequest) (*api.DeleteResponse, error) {
	txn := p.db.NewTransaction(false)
	defer txn.Discard()
	for _, key := range r.Keys {
		if err := txn.Delete([]byte(key)); err != nil {
			return nil, err
		}
	}
	return &api.DeleteResponse{}, nil
}

func (p *GeoDB) Stream(r *api.StreamRequest, ss api.GeoDB_StreamServer) error {
	clientID := p.hub.AddMessageStreamClient()
	for {
		select {
		case msg := <-p.hub.GetClientMessageStream(clientID):
			if err := ss.Send(&api.StreamResponse{
				Data: msg,
			}); err != nil {
				log.Error(err.Error())
			}
		case <-ss.Context().Done():
			p.hub.RemoveMessageStreamClient(clientID)
			break
		}
	}
}
