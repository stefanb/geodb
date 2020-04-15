package db

import (
	api "github.com/autom8ter/geodb/gen/go/geodb"
	"github.com/dgraph-io/badger/v2"
	"github.com/gogo/protobuf/proto"
	geo "github.com/paulmach/go.geo"
	"github.com/thoas/go-funk"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"regexp"
)

func ScanBound(db *badger.DB, bound *api.Bound, keys []string) (map[string]*api.ObjectDetail, error) {
	geoBound := geo.NewGeoBoundAroundPoint(geo.NewPointFromLatLng(bound.Center.Lat, bound.Center.Lon), bound.Radius)
	txn := db.NewTransaction(false)
	defer txn.Discard()
	objects := map[string]*api.ObjectDetail{}
	iter := txn.NewIterator(badger.DefaultIteratorOptions)
	defer iter.Close()
	for iter.Rewind(); iter.Valid(); iter.Next() {
		item := iter.Item()
		if item.UserMeta() != 1 {
			continue
		}
		if len(keys) > 0 {
			if funk.ContainsString(keys, string(item.Key())) {
				res, err := item.ValueCopy(nil)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to copy data: %s", err.Error())
				}
				if len(res) > 0 {
					var obj = &api.ObjectDetail{}
					if err := proto.Unmarshal(res, obj); err != nil {
						return nil, status.Errorf(codes.Internal, "failed to unmarshal protobuf: %s", err.Error())
					}
					if geoBound.Contains(geo.NewPointFromLatLng(obj.Object.Point.Lat, obj.Object.Point.Lon)) {
						objects[string(item.Key())] = obj
					}
				}
			}
		} else {
			res, err := item.ValueCopy(nil)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to copy data: %s", err.Error())
			}
			if len(res) > 0 {
				var obj = &api.ObjectDetail{}
				if err := proto.Unmarshal(res, obj); err != nil {
					return nil, status.Errorf(codes.Internal, "(all) %s failed to unmarshal protobuf: %s", string(item.Key()), err.Error())
				}
				if geoBound.Contains(geo.NewPointFromLatLng(obj.Object.Point.Lat, obj.Object.Point.Lon)) {
					objects[string(item.Key())] = obj
				}
			}
		}
	}
	return objects, nil
}

func ScanRegexBound(db *badger.DB, bound *api.Bound, rgex string) (map[string]*api.ObjectDetail, error) {
	geoBound := geo.NewGeoBoundAroundPoint(geo.NewPointFromLatLng(bound.Center.Lat, bound.Center.Lon), bound.Radius)
	txn := db.NewTransaction(false)
	defer txn.Discard()
	objects := map[string]*api.ObjectDetail{}
	iter := txn.NewIterator(badger.DefaultIteratorOptions)
	defer iter.Close()
	for iter.Rewind(); iter.Valid(); iter.Next() {
		item := iter.Item()
		if item.UserMeta() != 1 {
			continue
		}
		match, err := regexp.MatchString(rgex, string(item.Key()))
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
			if geoBound.Contains(geo.NewPointFromLatLng(obj.Object.Point.Lat, obj.Object.Point.Lon)) {
				objects[string(item.Key())] = obj
			}
		}
	}
	return objects, nil
}

func ScanPrefixBound(db *badger.DB, bound *api.Bound, prefix string) (map[string]*api.ObjectDetail, error) {
	geoBound := geo.NewGeoBoundAroundPoint(geo.NewPointFromLatLng(bound.Center.Lat, bound.Center.Lon), bound.Radius)
	txn := db.NewTransaction(false)
	defer txn.Discard()
	objects := map[string]*api.ObjectDetail{}
	iter := txn.NewIterator(badger.DefaultIteratorOptions)
	defer iter.Close()
	for iter.Seek([]byte(prefix)); iter.ValidForPrefix([]byte(prefix)); iter.Next() {
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
		if geoBound.Contains(geo.NewPointFromLatLng(obj.Object.Point.Lat, obj.Object.Point.Lon)) {
			objects[string(item.Key())] = obj
		}
	}
	return objects, nil
}
