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
	geoDB      *services.GeoDB
	coorsField = &api.Point{
		Lat: 39.756378173828125,
		Lon: -104.99414825439453,
	}
	pepsiCenter = &api.Point{
		Lat: 39.74863815307617,
		Lon: -105.00762176513672,
	}
	cherryCreekMall = &api.Point{
		Lat: 39.71670913696289,
		Lon: -104.95344543457031,
	}
	saintJosephHospital = &api.Point{
		Lat: 39.74626922607422,
		Lon: -104.97151184082031,
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

func TestPing(t *testing.T) {
	resp, err := geoDB.Ping(context.Background(), &api.PingRequest{})
	if err != nil {
		t.Fatal(err.Error())
	}
	if !resp.Ok {
		t.Fatal("expected ok ping")
	}
}

func TestSet(t *testing.T) {
	objects := []*api.Object{
		{
			Key:    "testing_coors",
			Point:  coorsField,
			Radius: 100,
			Tracking: &api.ObjectTracking{
				TravelMode: api.TravelMode_Driving,
			},
			Metadata:    nil,
			GetAddress:  true,
			GetTimezone: true,
			ExpiresUnix: 0,
		},
		{

			Key:    "testing_pepsi_center",
			Point:  pepsiCenter,
			Radius: 100,
			Tracking: &api.ObjectTracking{
				TravelMode: api.TravelMode_Driving,
				Trackers: []*api.ObjectTracker{
					{
						TargetObjectKey: "testing_coors",
						TrackDirections: true,
						TrackDistance:   true,
						TrackEta:        true,
					},
				},
			},
			GetAddress:  true,
			GetTimezone: true,
			ExpiresUnix: time.Now().Add(5 * time.Minute).Unix(),
		},
		{

			Key:    "testing_pepsi_center",
			Point:  pepsiCenter,
			Radius: 100,
			Tracking: &api.ObjectTracking{
				TravelMode: api.TravelMode_Driving,
				Trackers: []*api.ObjectTracker{
					{
						TargetObjectKey: "testing_coors",
						TrackDirections: true,
						TrackDistance:   true,
						TrackEta:        true,
					},
				},
			},
			GetAddress:  true,
			GetTimezone: true,
			ExpiresUnix: time.Now().Add(5 * time.Minute).Unix(),
		},
		{
			Key:    "malls_cherry_creek_mall",
			Point:  cherryCreekMall,
			Radius: 100,
			Tracking: &api.ObjectTracking{
				TravelMode: api.TravelMode_Driving,
				Trackers: []*api.ObjectTracker{
					{
						TargetObjectKey: "testing_pepsi_center",
						TrackDirections: true,
						TrackDistance:   true,
						TrackEta:        true,
					},
				},
			},
			GetAddress:  true,
			GetTimezone: true,
			ExpiresUnix: time.Now().Add(5 * time.Minute).Unix(),
		},
	}
	for _, obj := range objects {
		resp, err := geoDB.Set(context.Background(), &api.SetRequest{
			Objects: obj,
		})
		if err != nil {
			t.Fatal(err.Error())
		}
		t.Log(helpers.PrettyJson(resp))
	}
}

func TestGet(t *testing.T) {
	resp, err := geoDB.Get(context.Background(), &api.GetRequest{})
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(resp.Objects) != 3 {
		t.Fatal("expected 3 results")
	}
	for _, obj := range resp.Objects {
		t.Log(helpers.PrettyJson(obj))
	}
}

func TestGetPrefix(t *testing.T) {
	resp, err := geoDB.GetPrefix(context.Background(), &api.GetPrefixRequest{
		Prefix: "testing_",
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(resp.Objects) != 2 {
		t.Fatal("expected 2 results")
	}
	resp, err = geoDB.GetPrefix(context.Background(), &api.GetPrefixRequest{
		Prefix: "malls_",
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(resp.Objects) != 1 {
		t.Fatal("expected 1 results")
	}
}

func TestGetRegexs(t *testing.T) {
	resp, err := geoDB.GetRegex(context.Background(), &api.GetRegexRequest{
		Regex: "malls_*",
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(resp.Objects) != 1 {
		t.Fatal("expected 1 results")
	}
}

func TestGetKeys(t *testing.T) {
	resp, err := geoDB.GetKeys(context.Background(), &api.GetKeysRequest{})
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(resp.Keys) != 3 {
		t.Fatal("expected 3 results")
	}
}

func TestGetPrefixKeys(t *testing.T) {
	resp, err := geoDB.GetPrefixKeys(context.Background(), &api.GetPrefixKeysRequest{
		Prefix: "malls_",
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(resp.Keys) != 1 {
		t.Fatal("expected 1 results")
	}
	resp, err = geoDB.GetPrefixKeys(context.Background(), &api.GetPrefixKeysRequest{
		Prefix: "testing_",
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(resp.Keys) != 2 {
		t.Fatal("expected 2 results")
	}
}

func TestGetRegexKeys(t *testing.T) {
	resp, err := geoDB.GetRegexKeys(context.Background(), &api.GetRegexKeysRequest{
		Regex: "malls_*",
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(resp.Keys) != 1 {
		t.Fatal("expected 1 results")
	}
}

func TestScanBounds(t *testing.T) {
	resp, err := geoDB.ScanBound(context.Background(), &api.ScanBoundRequest{
		Bound: &api.Bound{
			Corner:         coorsField,
			OppositeCorner: saintJosephHospital,
		},
		Keys: nil,
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(resp.Objects) != 1 {
		t.Fatal("expected 1 results")
	}
}

func TestDelete(t *testing.T) {
	_, err := geoDB.Delete(context.Background(), &api.DeleteRequest{
		Keys: []string{"testing_pepsi_center"},
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	resp, err := geoDB.GetPrefixKeys(context.Background(), &api.GetPrefixKeysRequest{
		Prefix: "testing_",
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(resp.Keys) != 1 {
		t.Fatal("expected 1 results")
	}
}

func TestDeleteAll(t *testing.T) {
	_, err := geoDB.Delete(context.Background(), &api.DeleteRequest{
		Keys: []string{"*"},
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	resp, err := geoDB.Get(context.Background(), &api.GetRequest{})
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(resp.Objects) != 0 {
		t.Fatal("expected 0 results")
	}
}
