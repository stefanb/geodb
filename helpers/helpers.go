package helpers

import (
	api "github.com/autom8ter/geodb/gen/go/geodb"
	"googlemaps.github.io/maps"
)

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
