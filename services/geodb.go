package services

import (
	"context"
	api "github.com/autom8ter/geodb/gen/go/geodb"
	"github.com/autom8ter/geodb/helpers"
	"github.com/autom8ter/geodb/maps"
	"github.com/autom8ter/geodb/stream"
	"github.com/dgraph-io/badger/v2"
	"github.com/gogo/protobuf/proto"
	geo "github.com/paulmach/go.geo"
	log "github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"regexp"
	"time"
)

type GeoDB struct {
	hub   *stream.Hub
	db    *badger.DB
	gmaps *maps.Client
}

func NewGeoDB(db *badger.DB, hub *stream.Hub, gmaps *maps.Client) *GeoDB {
	return &GeoDB{
		hub:   hub,
		db:    db,
		gmaps: gmaps,
	}
}

func (p *GeoDB) Ping(ctx context.Context, req *api.PingRequest) (*api.PingResponse, error) {
	return &api.PingResponse{
		Ok: true,
	}, nil
}

func (p *GeoDB) Set(ctx context.Context, r *api.SetRequest) (*api.SetResponse, error) {
	var objects = map[string]*api.ObjectDetail{}
	for key, val := range r.Object {
		txn := p.db.NewTransaction(true)
		defer txn.Discard()
		val.Key = key
		if val.UpdatedUnix == 0 {
			val.UpdatedUnix = time.Now().Unix()
		}
		point1 := geo.NewPointFromLatLng(val.Point.Lat, val.Point.Lon)
		var events = map[string]*api.Event{}
		if len(val.Trackers) > 0 {
			for _, tracker := range val.Trackers {
				item, err := txn.Get([]byte(tracker))
				if err != nil {
					log.Error(err.Error())
					continue
				}
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
				event := &api.Event{
					Object:        obj,
					Distance:      dist,
					Inside:        dist <= float64(val.Radius+obj.Radius),
					TimestampUnix: val.UpdatedUnix,
				}
				if p.gmaps != nil {
					directions, eta, dist, err := p.gmaps.TravelDetail(context.Background(), val.Point, obj.Point, helpers.ToTravelMode(val.TravelMode))
					if err != nil {
						log.Error(err.Error())
					} else {
						event.Direction = &api.Directions{
							HtmlDirections: directions,
							Eta:            int64(eta),
							TravelDist:     int64(dist),
						}
					}
				}
				events[obj.Key] = event
			}
		}
		detail := &api.ObjectDetail{
			Object: val,
		}
		if p.gmaps != nil {
			addr, err := p.gmaps.GetAddress(val.Point)
			if err != nil {
				log.Error(err.Error())
			}
			detail.Address = addr
		}
		if len(events) > 0 {
			for _, event := range events {
				detail.Events = append(detail.Events, event)
			}
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
		objects[detail.Object.Key] = detail
	}
	return &api.SetResponse{
		Object: objects,
	}, nil
}

func (p *GeoDB) GetRegex(ctx context.Context, r *api.GetRegexRequest) (*api.GetRegexResponse, error) {
	txn := p.db.NewTransaction(false)
	defer txn.Discard()
	objects := map[string]*api.ObjectDetail{}
	iter := txn.NewIterator(badger.DefaultIteratorOptions)
	for iter.Rewind(); iter.Valid(); iter.Next() {
		item := iter.Item()
		if item.UserMeta() != 1 {
			continue
		}
		match, err := regexp.MatchString(r.Regex, string(item.Key()))
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to match regex: %s", err.Error())
		}
		if match {
			res, err := item.ValueCopy(nil)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to copy data: %s", err.Error())
			}
			var obj = &api.ObjectDetail{}
			if err := proto.Unmarshal(res, obj); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to unmarshal protobuf: %s", err.Error())
			}
			objects[string(item.Key())] = obj
		}

	}
	iter.Close()
	return &api.GetRegexResponse{
		Object: objects,
	}, nil
}

func (p *GeoDB) Get(ctx context.Context, r *api.GetRequest) (*api.GetResponse, error) {
	txn := p.db.NewTransaction(false)
	defer txn.Discard()
	objects := map[string]*api.ObjectDetail{}
	if len(r.Keys) == 0 {
		iter := txn.NewIterator(badger.DefaultIteratorOptions)
		for iter.Rewind(); iter.Valid(); iter.Next() {
			item := iter.Item()
			res, err := item.ValueCopy(nil)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to copy data: %s", err.Error())
			}
			if len(res) > 0 {
				var obj = &api.ObjectDetail{}
				if err := proto.Unmarshal(res, obj); err != nil {
					return nil, status.Errorf(codes.Internal, "failed to unmarshal protobuf: %s", err.Error())
				}
				objects[string(item.Key())] = obj
			}
		}
		iter.Close()
	} else {
		for _, key := range r.Keys {
			i, err := txn.Get([]byte(key))
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "failed to get key: %s", err.Error())
			}
			if i.UserMeta() != 1 {
				continue
			}
			res, err := i.ValueCopy(nil)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to copy data: %s", err.Error())
			}
			var obj = &api.ObjectDetail{}
			if err := proto.Unmarshal(res, obj); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to unmarshal protobuf: %s", err.Error())
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
	objects := map[string]*api.ObjectDetail{}
	iter := txn.NewIterator(badger.DefaultIteratorOptions)
	for iter.Seek([]byte(r.Prefix)); iter.ValidForPrefix([]byte(r.Prefix)); iter.Next() {
		item := iter.Item()
		if item.UserMeta() != 1 {
			continue
		}
		res, err := item.ValueCopy(nil)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to copy data: %s", err.Error())
		}
		var obj = &api.ObjectDetail{}
		if err := proto.Unmarshal(res, obj); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal protobuf: %s", err.Error())
		}
		objects[string(item.Key())] = obj
	}
	iter.Close()
	return &api.SeekResponse{
		Object: objects,
	}, nil
}

func (p *GeoDB) GetKeys(ctx context.Context, r *api.GetKeysRequest) (*api.GetKeysResponse, error) {
	txn := p.db.NewTransaction(false)
	defer txn.Discard()
	keys := []string{}
	iter := txn.NewIterator(badger.DefaultIteratorOptions)
	for iter.Rewind(); iter.Valid(); iter.Next() {
		item := iter.Item()
		if item.UserMeta() != 1 {
			continue
		}
		keys = append(keys, string(item.Key()))
	}
	iter.Close()
	return &api.GetKeysResponse{
		Keys: keys,
	}, nil
}

func (p *GeoDB) Delete(ctx context.Context, r *api.DeleteRequest) (*api.DeleteResponse, error) {
	txn := p.db.NewTransaction(true)
	defer txn.Discard()
	if len(r.Keys) > 0 && r.Keys[0] == "*" {
		iter := txn.NewIterator(badger.DefaultIteratorOptions)
		for iter.Rewind(); iter.Valid(); iter.Next() {
			item := iter.Item()
			if err := txn.Delete(item.Key()); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to delete key: %s %s", string(item.Key()), err.Error())
			}
		}
		iter.Close()
	} else {
		for _, key := range r.Keys {
			if err := txn.Delete([]byte(key)); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to delete key: %s %s", key, err.Error())
			}
		}
	}
	if err := txn.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete keys %s", err.Error())
	}
	return &api.DeleteResponse{}, nil
}

func (p *GeoDB) StreamRegex(r *api.StreamRegexRequest, ss api.GeoDB_StreamRegexServer) error {
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
					if err := ss.Send(&api.StreamRegexResponse{
						Object: msg,
					}); err != nil {
						log.Error(err.Error())
					}
				}
			} else {
				if err := ss.Send(&api.StreamRegexResponse{
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

func (p *GeoDB) Stream(r *api.StreamRequest, ss api.GeoDB_StreamServer) error {
	clientID := p.hub.AddObjectStreamClient(r.ClientId)
	for {
		select {
		case msg := <-p.hub.GetClientObjectStream(clientID):
			if len(r.Keys) > 0 {
				if funk.ContainsString(r.Keys, msg.Object.Key) {
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
