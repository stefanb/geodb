package maps

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	api "github.com/autom8ter/geodb/gen/go/geodb"
	"github.com/dgraph-io/badger/v2"
	"github.com/gogo/protobuf/proto"
	geo "github.com/paulmach/go.geo"
	"googlemaps.github.io/maps"
	"strings"
	"time"
)

type Client struct {
	googleMapsClient     *maps.Client
	db                   *badger.DB
	precision            int
	directionsExpiration time.Duration
}

const (
	directionsMeta  = 2
	timezoneMeta    = 3
	addressMeta     = 4
	coordinatesMeta = 5
)

func NewClient(db *badger.DB, apiKey string, directionsExpiration time.Duration) (*Client, error) {
	client, err := maps.NewClient(maps.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	return &Client{
		googleMapsClient:     client,
		db:                   db,
		directionsExpiration: directionsExpiration,
	}, nil
}

func (c *Client) Directions(ctx context.Context, origin *api.Point, dest *api.Point, mode maps.Mode) ([]maps.Route, error) {
	res, err := c.getCachedDirections(origin, dest, mode)
	if err != nil {
		return nil, err
	}
	if res != nil && len(res.Routes) > 0 {
		return res.Routes, nil
	}
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
	if err := c.cacheDirections(origin, dest, mode, &RouteCache{
		Routes: resp,
	}); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) GetAddress(point *api.Point) (*api.Address, error) {
	addr, err := c.getCachedAddress(point)
	if err != nil {
		return nil, err
	}
	if addr != nil && addr.Address != "" {
		return addr, nil
	}
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
	if err := c.cacheAddress(point, address); err != nil {
		return nil, err
	}
	return address, nil
}

func (c *Client) GetTimezone(point *api.Point) (string, error) {
	zone, err := c.getCachedTimezone(point)
	if err != nil {
		return "", err
	}
	if zone != "" {
		return zone, nil
	}
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
		return "", err
	}
	if err := c.cacheTimezone(point, timezoneResult.TimeZoneID); err != nil {
		return "", err
	}
	return timezoneResult.TimeZoneID, nil
}

func (c *Client) PointString(point *api.Point) string {
	return fmt.Sprintf("%f, %f", point.Lat, point.Lon)
}

func (c *Client) TravelDetail(ctx context.Context, here, there *api.Point, mode maps.Mode) (string, int, int, error) {
	directions, err := c.Directions(ctx, here, there, mode)
	if err != nil {
		return "", 0, 0, err
	}
	htmlDirections := fmt.Sprintf("\n<h5>Destination: %s</h5>", directions[0].Legs[len(directions[0].Legs)-1].EndAddress)
	eta := 0
	dist := 0
	for _, leg := range directions[0].Legs {
		eta += int(leg.DurationInTraffic.Minutes())
		dist += leg.Meters
		for _, step := range leg.Steps {
			htmlDirections += fmt.Sprintf("%s - %s", step.HTMLInstructions, step.HumanReadable)
			htmlDirections += "<br>"
		}
	}
	return base64.StdEncoding.EncodeToString([]byte(htmlDirections)), eta, dist, nil
}

func (c *Client) GetCoordinates(address string) (*api.Point, error) {
	point, err := c.getCachedCoordinates(address)
	if err != nil {
		return nil, err
	}
	if point != nil {
		return point, nil
	}
	req := &maps.GeocodingRequest{
		Address: address,
	}
	resp, err := c.googleMapsClient.Geocode(context.Background(), req)
	if err != nil {
		return &api.Point{}, err
	}
	point = &api.Point{
		Lon: resp[0].Geometry.Location.Lng,
		Lat: resp[0].Geometry.Location.Lat,
	}
	if err := c.cacheCoordinates(address, point); err != nil {
		return nil, err
	}
	return point, nil
}

type RouteCache struct {
	Routes []maps.Route `json:"routes"`
}

func (c *Client) cacheDirections(origin, destination *api.Point, mode maps.Mode, routes *RouteCache) error {
	orig, dest := geo.NewPointFromLatLng(origin.Lat, origin.Lon), geo.NewPointFromLatLng(destination.Lat, destination.Lon)
	tx := c.db.NewTransaction(true)
	defer tx.Discard()
	bits, err := json.Marshal(routes)
	if err != nil {
		return err
	}
	if err := tx.SetEntry(&badger.Entry{
		Key:       []byte(c.directionsCacheKey(orig, dest, mode)),
		Value:     bits,
		UserMeta:  directionsMeta,
		ExpiresAt: uint64(time.Now().Add(c.directionsExpiration).Unix()),
	}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (c *Client) getCachedDirections(origin, destination *api.Point, mode maps.Mode) (*RouteCache, error) {
	tx := c.db.NewTransaction(false)
	defer tx.Discard()
	orig, dest := geo.NewPointFromLatLng(origin.Lat, origin.Lon), geo.NewPointFromLatLng(destination.Lat, destination.Lon)
	item, err := tx.Get([]byte(c.directionsCacheKey(orig, dest, mode)))
	if err != nil {
		return nil, err
	}
	res, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err
	}
	if len(res) > 0 {
		var routes = &RouteCache{}
		if err := json.Unmarshal(res, routes); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (c *Client) cacheAddress(point *api.Point, addr *api.Address) error {
	if addr == nil {
		return nil
	}
	gpoint := geo.NewPointFromLatLng(point.Lat, point.Lon)
	tx := c.db.NewTransaction(true)
	defer tx.Discard()
	bits, err := proto.Marshal(addr)
	if err != nil {
		return err
	}
	if err := tx.SetEntry(&badger.Entry{
		Key:       []byte(c.addressCacheKey(gpoint)),
		Value:     bits,
		UserMeta:  addressMeta,
		ExpiresAt: 0,
	}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (c *Client) getCachedAddress(point *api.Point) (*api.Address, error) {
	gpoint := geo.NewPointFromLatLng(point.Lat, point.Lon)
	tx := c.db.NewTransaction(false)
	defer tx.Discard()
	item, err := tx.Get([]byte(c.addressCacheKey(gpoint)))
	if err != nil {
		return nil, err
	}
	res, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err
	}
	if len(res) > 0 {
		var routes = &api.Address{}
		if err := proto.Unmarshal(res, routes); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (c *Client) cacheTimezone(point *api.Point, zone string) error {
	gpoint := geo.NewPointFromLatLng(point.Lat, point.Lon)
	tx := c.db.NewTransaction(true)
	defer tx.Discard()
	e := &badger.Entry{
		Key:       []byte(c.timezoneCacheKey(gpoint)),
		Value:     []byte(zone),
		UserMeta:  timezoneMeta,
		ExpiresAt: 0,
	}

	if err := tx.SetEntry(e); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (c *Client) getCachedTimezone(point *api.Point) (string, error) {
	gpoint := geo.NewPointFromLatLng(point.Lat, point.Lon)
	tx := c.db.NewTransaction(false)
	defer tx.Discard()
	item, err := tx.Get([]byte(c.timezoneCacheKey(gpoint)))
	if err != nil {
		return "", err
	}
	res, err := item.ValueCopy(nil)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

func (c *Client) cacheCoordinates(address string, point *api.Point) error {
	tx := c.db.NewTransaction(true)
	defer tx.Discard()
	bits, err := proto.Marshal(point)
	if err != nil {
		return err
	}
	e := &badger.Entry{
		Key:       []byte(c.coordinatesCacheKey(address)),
		Value:     []byte(bits),
		UserMeta:  coordinatesMeta,
		ExpiresAt: 0,
	}

	if err := tx.SetEntry(e); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (c *Client) getCachedCoordinates(address string) (*api.Point, error) {
	tx := c.db.NewTransaction(false)
	defer tx.Discard()
	item, err := tx.Get([]byte(c.coordinatesCacheKey(address)))
	if err != nil {
		return nil, err
	}
	res, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err
	}
	if len(res) > 0 {
		var point = &api.Point{}
		if err := proto.Unmarshal(res, point); err != nil {
			return nil, err
		}
		return point, nil
	}
	return nil, nil
}

func (c *Client) directionsCacheKey(origin, destination *geo.Point, mode maps.Mode) string {
	originHash := origin.GeoHash(9)
	destHash := destination.GeoHash(c.precision)
	return fmt.Sprintf("gmaps_directions_%s_%s_%s", mode, originHash, destHash)
}

func (c *Client) addressCacheKey(point *geo.Point) string {
	hash := point.GeoHash(9)
	return fmt.Sprintf("gmaps_address_%s", hash)
}

func (c *Client) timezoneCacheKey(point *geo.Point) string {
	hash := point.GeoHash(4)
	return fmt.Sprintf("gmaps_timezone_%s", hash)
}

func (c *Client) coordinatesCacheKey(address string) string {
	return fmt.Sprintf("gmaps_coordinates_%s", base64.StdEncoding.EncodeToString([]byte(strings.ToLower(strings.TrimSpace(address)))))
}
