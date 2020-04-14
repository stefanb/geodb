package helpers

import (
	"fmt"
	api "github.com/autom8ter/geodb/gen/go/geodb"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
	"googlemaps.github.io/maps"
)

var jpb = jsonpb.Marshaler{}

func ToTravelMode(mode api.TravelMode) maps.Mode {
	switch mode {
	case api.TravelMode_Bicycling:
		return maps.TravelModeBicycling
	case api.TravelMode_Transit:
		return maps.TravelModeTransit
	case api.TravelMode_Walking:
		return maps.TravelModeWalking
	default:
		return maps.TravelModeDriving
	}
}

func PrettyJson(msg proto.Message) string {
	str, _ := jpb.MarshalToString(msg)
	return fmt.Sprintln(str)
}
