package geofence

import (
	api "github.com/autom8ter/geodb/gen/go/geodb"
	"github.com/autom8ter/geodb/stream"
	"github.com/dgraph-io/badger/v2"
	"github.com/gogo/protobuf/proto"
	geo "github.com/paulmach/go.geo"
	log "github.com/sirupsen/logrus"
)

func Geofence(db *badger.DB, trigger *api.Object) {
	txn2 := db.NewTransaction(false)
	defer txn2.Discard()
	point1 := geo.NewPointFromLatLng(trigger.Point.Lat, trigger.Point.Lon)
	iter := txn2.NewIterator(badger.DefaultIteratorOptions)
	var events = map[string]*api.Event{}
	for iter.Rewind(); iter.Valid(); iter.Next() {
		item := iter.Item()
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
		if obj.Key != trigger.Key {
			point2 := geo.NewPointFromLatLng(obj.Point.Lat, obj.Point.Lon)
			dist := point1.GeoDistanceFrom(point2, true)
			if dist <= float64(trigger.Radius+obj.Radius) {
				events[obj.Key] = &api.Event{
					TriggerObject: trigger,
					Object:        obj,
					Distance:      dist,
					TimestampUnix: trigger.UpdatedUnix,
				}
			}
		}

	}
	iter.Close()
	for _, event := range events {
		stream.PublishEvent(event)
	}
}
