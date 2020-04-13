package services

import (
	"context"
	api "github.com/autom8ter/geodb/gen/go/geodb"
	"github.com/autom8ter/geodb/stream"
	"github.com/dgraph-io/badger/v2"
	"github.com/gogo/protobuf/proto"
	geo "github.com/paulmach/go.geo"
	log "github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
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

func (p *GeoDB) Set(ctx context.Context, r *api.SetRequest) (*api.SetResponse, error) {
	for key, val := range r.Object {
		txn := p.db.NewTransaction(true)
		defer txn.Discard()
		val.Key = key
		if val.UpdatedUnix == 0 {
			val.UpdatedUnix = time.Now().Unix()
		}
		point1 := geo.NewPointFromLatLng(val.Point.Lat, val.Point.Lon)
		iter := txn.NewIterator(badger.DefaultIteratorOptions)
		var events = map[string]*api.Event{}
		for iter.Rewind(); iter.Valid(); iter.Next() {
			item := iter.Item()
			if string(item.Key()) != val.Key && len(val.GeofenceTriggers) > 0 && funk.ContainsString(val.GeofenceTriggers, string(item.Key())) {
				res, err := item.ValueCopy(nil)
				if err != nil {
					log.Error(err.Error())
					continue
				}
				var obj = &api.Object{}
				if err := proto.Unmarshal(res, obj); err != nil {
					log.Error(err.Error())
					continue
				}
				if obj.Point == nil {
					continue
				}
				point2 := geo.NewPointFromLatLng(obj.Point.Lat, obj.Point.Lon)
				dist := point1.GeoDistanceFrom(point2, true)
				events[obj.Key] = &api.Event{
					Object:        obj,
					Distance:      dist,
					Inside:        dist <= float64(val.Radius+obj.Radius),
					TimestampUnix: val.UpdatedUnix,
				}
			}
		}
		iter.Close()

		detail := &api.ObjectDetail{
			Object: val,
		}
		for _, event := range events {
			detail.Events = append(detail.Events, event)
		}
		bits, _ := proto.Marshal(detail)
		if err := txn.SetEntry(&badger.Entry{
			Key:       []byte(key),
			Value:     bits,
			UserMeta:  1,
			ExpiresAt: uint64(val.ExpiresUnix),
		}); err != nil {
			log.Error(err.Error())
		}

		if err := txn.Commit(); err != nil {
			log.Error(err.Error())
			continue
		}
		p.hub.PublishObject(detail)
	}
	return &api.SetResponse{}, nil
}

func (p *GeoDB) GetRegex(ctx context.Context, r *api.GetRegexRequest) (*api.GetRegexResponse, error) {
	txn := p.db.NewTransaction(false)
	defer txn.Discard()
	objects := map[string]*api.Object{}
	iter := txn.NewIterator(badger.DefaultIteratorOptions)
	defer iter.Close()
	for iter.Rewind(); iter.Valid(); iter.Next() {
		item := iter.Item()
		if item.UserMeta() != 1 {
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
	return &api.GetRegexResponse{
		Object: objects,
	}, nil
}

func (p *GeoDB) Get(ctx context.Context, r *api.GetRequest) (*api.GetResponse, error) {
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
			if i.UserMeta() != 1 {
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
	return &api.GetResponse{
		Object: objects,
	}, nil
}

func (p *GeoDB) Seek(ctx context.Context, r *api.SeekRequest) (*api.SeekResponse, error) {
	txn := p.db.NewTransaction(false)
	defer txn.Discard()
	objects := map[string]*api.Object{}
	iter := txn.NewIterator(badger.DefaultIteratorOptions)
	defer iter.Close()
	for iter.Seek([]byte(r.Prefix)); iter.ValidForPrefix([]byte(r.Prefix)); iter.Next() {
		item := iter.Item()
		if item.UserMeta() != 1 {
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
	return &api.SeekResponse{
		Object: objects,
	}, nil
}

func (p *GeoDB) Delete(ctx context.Context, r *api.DeleteRequest) (*api.DeleteResponse, error) {
	txn := p.db.NewTransaction(true)
	defer txn.Discard()
	for _, key := range r.Keys {
		if err := txn.Delete([]byte(key)); err != nil {
			return nil, err
		}
	}
	return &api.DeleteResponse{}, nil
}

func (p *GeoDB) Stream(r *api.StreamRequest, ss api.GeoDB_StreamServer) error {
	clientID := p.hub.AddObjectStreamClient(r.ClientId)
	for {
		select {
		case msg := <-p.hub.GetClientObjectStream(clientID):
			if r.Regex != "" {
				match, err := regexp.MatchString(r.Regex, msg.Object.Key)
				if err != nil {
					return err
				}
				if match {
					if err := ss.Send(&api.StreamResponse{
						Object: msg,
					}); err != nil {
						log.Error(err.Error())
					}
				}
			} else {
				if err := ss.Send(&api.StreamResponse{
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
