package services

import (
	"context"
	api "github.com/autom8ter/geodb/gen/go/geodb"
	"github.com/autom8ter/geodb/geofence"
	"github.com/autom8ter/geodb/meta"
	"github.com/autom8ter/geodb/stream"
	"github.com/dgraph-io/badger/v2"
	"github.com/gogo/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"regexp"
	"time"
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

func (p *GeoDB) SetObject(ctx context.Context, r *api.SetObjectRequest) (*api.SetObjectResponse, error) {
	for key, val := range r.Object {
		txn := p.db.NewTransaction(true)
		defer txn.Discard()
		val.Key = key
		if val.UpdatedUnix == 0 {
			val.UpdatedUnix = time.Now().Unix()
		}
		bits, _ := proto.Marshal(val)
		e := &badger.Entry{
			Key:       []byte(key),
			Value:     bits,
			UserMeta:  meta.ObjectMeta.Byte(),
			ExpiresAt: uint64(val.ExpiresUnix),
		}
		if err := txn.SetEntry(e); err != nil {
			log.Error(err.Error())
			continue
		}
		p.hub.PublishObject(val)
		if err := txn.Commit(); err != nil {
			log.Error(err.Error())
			continue
		}
		go func(obj *api.Object) {
			geofence.Geofence(p.db, obj)
		}(val)
	}
	return &api.SetObjectResponse{}, nil
}

func (p *GeoDB) GetObjectRegex(ctx context.Context, r *api.GetObjectRegexRequest) (*api.GetObjectRegexResponse, error) {
	txn := p.db.NewTransaction(false)
	defer txn.Discard()
	objects := map[string]*api.Object{}
	iter := txn.NewIterator(badger.DefaultIteratorOptions)
	defer iter.Close()
	for iter.Rewind(); iter.Valid(); iter.Next() {
		item := iter.Item()
		if item.UserMeta() != meta.ObjectMeta.Byte() {
			continue
		}
		match, err := regexp.MatchString(r.Regex, string(item.Key()))
		if err != nil {
			return nil, err
		}
		if match {
			res, err := item.ValueCopy(nil)
			if err != nil {
				return nil, err
			}
			var obj = &api.Object{}
			if err := proto.Unmarshal(res, obj); err != nil {
				return nil, err
			}
			objects[string(item.Key())] = obj
		}

	}
	return &api.GetObjectRegexResponse{
		Object: objects,
	}, nil
}

func (p *GeoDB) GetObject(ctx context.Context, r *api.GetObjectRequest) (*api.GetObjectResponse, error) {
	txn := p.db.NewTransaction(false)
	defer txn.Discard()
	objects := map[string]*api.Object{}
	if len(r.Keys) > 0 && r.Keys[0] == "*" {
		iter := txn.NewIterator(badger.DefaultIteratorOptions)
		defer iter.Close()
		for iter.Rewind(); iter.Valid(); iter.Next() {
			item := iter.Item()
			res, err := item.ValueCopy(nil)
			if err != nil {
				return nil, err
			}
			var obj = &api.Object{}
			if err := proto.Unmarshal(res, obj); err != nil {
				return nil, err
			}
			objects[string(item.Key())] = obj
		}
	} else {
		for _, key := range r.Keys {
			i, err := txn.Get([]byte(key))
			if err != nil {
				return nil, err
			}
			if i.UserMeta() != meta.ObjectMeta.Byte() {
				continue
			}
			res, err := i.ValueCopy(nil)
			if err != nil {
				return nil, err
			}
			var obj = &api.Object{}
			if err := proto.Unmarshal(res, obj); err != nil {
				return nil, err
			}
			objects[key] = obj
		}
	}
	return &api.GetObjectResponse{
		Object: objects,
	}, nil
}

func (p *GeoDB) SeekObject(ctx context.Context, r *api.SeekObjectRequest) (*api.SeekObjectResponse, error) {
	txn := p.db.NewTransaction(false)
	defer txn.Discard()
	objects := map[string]*api.Object{}
	iter := txn.NewIterator(badger.DefaultIteratorOptions)
	defer iter.Close()
	for iter.Seek([]byte(r.Prefix)); iter.ValidForPrefix([]byte(r.Prefix)); iter.Next() {
		item := iter.Item()
		if item.UserMeta() != meta.ObjectMeta.Byte() {
			continue
		}
		res, err := item.ValueCopy(nil)
		if err != nil {
			return nil, err
		}
		var obj = &api.Object{}
		if err := proto.Unmarshal(res, obj); err != nil {
			return nil, err
		}
		objects[string(item.Key())] = obj
	}
	return &api.SeekObjectResponse{
		Object: objects,
	}, nil
}

func (p *GeoDB) DeleteObject(ctx context.Context, r *api.DeleteObjectRequest) (*api.DeleteObjectResponse, error) {
	txn := p.db.NewTransaction(true)
	defer txn.Discard()
	for _, key := range r.Keys {
		if err := txn.Delete([]byte(key)); err != nil {
			return nil, err
		}
	}
	return &api.DeleteObjectResponse{}, nil
}

func (p *GeoDB) StreamObject(r *api.StreamObjectRequest, ss api.GeoDB_StreamObjectServer) error {
	clientID := p.hub.AddObjectStreamClient(r.ClientId)
	for {
		select {
		case msg := <-p.hub.GetClientObjectStream(clientID):
			if r.Regex != "" {
				match, err := regexp.MatchString(r.Regex, msg.Key)
				if err != nil {
					return err
				}
				if match {
					if err := ss.Send(&api.StreamObjectResponse{
						Object: msg,
					}); err != nil {
						log.Error(err.Error())
					}
				}
			} else {
				if err := ss.Send(&api.StreamObjectResponse{
					Object: msg,
				}); err != nil {
					log.Error(err.Error())
				}
			}
		case <-ss.Context().Done():
			p.hub.RemoveObjectStreamClient(clientID)
			break
		}
	}
}

func (p *GeoDB) StreamEvents(r *api.StreamEventsRequest, ss api.GeoDB_StreamEventsServer) error {
	clientID := p.hub.AddEventStreamClient(r.ClientId)
	defer p.hub.RemoveEventStreamClient(clientID)
	for {
		select {
		case event := <-p.hub.GetClientEventStream(clientID):
			if event == nil {
				continue
			}
			if r.Regex != "" {
				match, err := regexp.MatchString(r.Regex, event.TriggerObject.Key)
				if err != nil {
					return err
				}
				if match {
					if err := ss.Send(&api.StreamEventsResponse{
						Events: event,
					}); err != nil {
						log.Error(err.Error())
					}
				}
			} else {
				if err := ss.Send(&api.StreamEventsResponse{
					Events: event,
				}); err != nil {
					log.Error(err.Error())
				}
			}
		case <-ss.Context().Done():
			break
		}
	}
}
