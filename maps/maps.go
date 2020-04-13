package maps

import (
	"context"
	"fmt"
	api "github.com/autom8ter/geodb/gen/go/geodb"
	"googlemaps.github.io/maps"
	"strings"
	"time"
)

type Client struct {
	googleMapsClient *maps.Client
}

func NewClient(googleMapsClient *maps.Client) *Client {
	return &Client{googleMapsClient: googleMapsClient}
}

func (c *Client) Directions(ctx context.Context, origin api.Point, dest api.Point, mode maps.Mode) ([]maps.Route, error) {
	resp, _, err := c.googleMapsClient.Directions(ctx, &maps.DirectionsRequest{
		Origin:        c.PointString(origin),
		Destination:   c.PointString(dest),
		Mode:          mode,
		DepartureTime: "now",
		TrafficModel:  maps.TrafficModelBestGuess,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) GetAddress(point *api.Point) (*api.Address, error) {
	location := &maps.LatLng{
		Lat: point.Lat,
		Lng: point.Lon,
	}

	req := &maps.GeocodingRequest{
		LatLng: location,
	}

	resp, err := c.googleMapsClient.ReverseGeocode(context.Background(), req)
	if err != nil {
		return nil, err
	}

	var address = &api.Address{}
	for _, res := range resp {
		address.Address = res.FormattedAddress
		for _, addressComponent := range res.AddressComponents {
			for _, t := range addressComponent.Types {
				switch t {
				case "administrative_area_level_1":
					longName := addressComponent.LongName
					address.State = longName
				case "administrative_area_level_2":
					address.County = addressComponent.LongName
				case "country":
					address.Country = addressComponent.LongName
				case "postal_code":
					address.Zip = addressComponent.LongName
				case "locality", "sublocality":
					address.City = addressComponent.LongName
				default:
					continue
				}
			}
		}
		break
	}
	return address, nil
}

func (c *Client) GetTimezone(point api.Point) (string, error) {
	timezoneID := "America/Chicago"
	location := &maps.LatLng{
		Lat: point.Lat,
		Lng: point.Lon,
	}

	r := &maps.TimezoneRequest{
		Location:  location,
		Timestamp: time.Now(),
	}
	timezoneResult, err := c.googleMapsClient.Timezone(context.Background(), r)
	if err != nil {
		return timezoneID, err
	}

	timezoneID = timezoneResult.TimeZoneID

	return timezoneID, nil
}

func (c *Client) PointString(point api.Point) string {
	return fmt.Sprintf("%f, %f", point.Lat, point.Lon)
}

func (c *Client) TravelDetail(ctx context.Context, here, there api.Point, mode maps.Mode) (string, int, int, error) {
	directions, err := c.Directions(ctx, here, there, mode)
	if err != nil {
		return "", 0, 0, err
	}
	directionBuilder := &strings.Builder{}
	eta := 0
	dist := 0
	for _, leg := range directions[0].Legs {
		eta += int(leg.DurationInTraffic.Minutes())
		dist += leg.Meters
		for _, step := range leg.Steps {
			directionBuilder.WriteString(fmt.Sprintln(step.HTMLInstructions))
		}
	}
	return directionBuilder.String(), eta, dist, nil
}

func (c *Client) GetCoordinates(address string) (*api.Point, error) {
	req := &maps.GeocodingRequest{
		Address: address,
	}

	resp, err := c.googleMapsClient.Geocode(context.Background(), req)
	if err != nil {
		return &api.Point{}, err
	}
	point := &api.Point{
		Lon: resp[0].Geometry.Location.Lng,
		Lat: resp[0].Geometry.Location.Lat,
	}
	return point, nil
}