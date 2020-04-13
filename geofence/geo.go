package geofence

import (
	"fmt"
	api "github.com/autom8ter/geodb/gen/go/geodb"
	"github.com/autom8ter/geodb/meta"
	"github.com/autom8ter/geodb/stream"
	"github.com/dgraph-io/badger/v2"
	"github.com/gogo/protobuf/proto"
	geo "github.com/paulmach/go.geo"
	log "github.com/sirupsen/logrus"
	"regexp"
)

const EventsPrefix = "events__"

func GetEventsPrefix(object *api.Object) []byte {
	return []byte(fmt.Sprintf("%s%s", EventsPrefix, object.Key))
}

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
			if trigger.GeofenceRegex != "" {
				match, err := regexp.MatchString(trigger.GeofenceRegex, obj.Key)
				if err != nil {
					log.Error(err.Error())
					continue
				}
				if match {
					point2 := geo.NewPointFromLatLng(obj.Point.Lat, obj.Point.Lon)
					dist := point1.GeoDistanceFrom(point2, true)
					if dist <= float64(trigger.Radius+obj.Radius) {
						events[obj.Key] = &api.Event{
							Object:        obj,
							Distance:      dist,
							TimestampUnix: trigger.UpdatedUnix,
						}
					}
				}
			} else {
				point2 := geo.NewPointFromLatLng(obj.Point.Lat, obj.Point.Lon)
				dist := point1.GeoDistanceFrom(point2, true)
				if dist <= float64(trigger.Radius+obj.Radius) {
					events[obj.Key] = &api.Event{
						Object:        obj,
						Distance:      dist,
						TimestampUnix: trigger.UpdatedUnix,
					}
				}
			}
		}
	}
	iter.Close()
	eventArr := &api.Events{
		TriggerObject: trigger,
	}
	for _, event := range events {
		eventArr.Events = append(eventArr.Events, event)
	}
	bits, _ := proto.Marshal(eventArr)
	if err := txn2.SetEntry(&badger.Entry{
		Key:       GetEventsPrefix(trigger),
		Value:     bits,
		UserMeta:  meta.EventMeta.Byte(),
		ExpiresAt: uint64(trigger.ExpiresUnix),
	}); err != nil {
		log.Error(err.Error())
		return
	}
	if err := txn2.Commit(); err != nil {
		log.Error(err.Error())
		return
	}
	stream.PublishEvent(eventArr)
}
