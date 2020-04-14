package main

import (
	"context"
	api "github.com/autom8ter/geodb/gen/go/geodb"
	"github.com/autom8ter/geodb/helpers"
	"github.com/autom8ter/geodb/server"
	"github.com/autom8ter/geodb/services"
	"log"
	"os"
	"testing"
	"time"
)


var (
	geoDB *services.GeoDB
	coorsField = &api.Point{
		Lat:                  39.756378173828125,
		Lon:                  -104.99414825439453,
	}
	pepsiCenter = &api.Point{
		Lat:                  39.74863815307617,
		Lon:                  -105.00762176513672,
	}
)

func TestMain(t *testing.M) {
	db, hub, gmaps, err := server.GetDeps()
	if err != nil {
		log.Fatal(err.Error())
	}
	geoDB = services.NewGeoDB(db, hub, gmaps)
	os.Exit(t.Run())
}

func Test(t *testing.T) {
	 resp, err := geoDB.Set(context.Background(), &api.SetRequest{
		Objects: map[string]*api.Object{
			"testing_coors" : &api.Object{
				Key:                  "testing_coors",
				Point:                coorsField,
				Radius:               100,
				Tracking:             &api.ObjectTracking{
					TravelMode:           api.TravelMode_Driving,
				},
				Metadata:             nil,
				GetAddress:           true,
				GetTimezone:          true,
				ExpiresUnix:          0,
			},
			"testing_pepsi_center" : &api.Object{
				Key:                  "testing_pepsi_center",
				Point:                pepsiCenter,
				Radius:               100,
				Tracking:             &api.ObjectTracking{
					TravelMode:           api.TravelMode_Driving,
					Trackers: []*api.ObjectTracker{
						{
							TargetObjectKey:      "testing_coors",
							TrackDirections:      true,
							TrackDistance:        true,
							TrackEta:             true,
						},
					},
				},
				GetAddress:           true,
				GetTimezone:          true,
				ExpiresUnix:          time.Now().Add(5 *time.Minute).Unix(),
			},
		},
	})
	 if err != nil {
	 	t.Fatal(err.Error())
	 }
	 for _, obj := range resp.Objects {
	 	t.Log(helpers.PrettyJson(obj))
	 }
}


