# GeoDB- A Persistent Geospatial Database

    go get github.com/autom8ter/geodb
    docker pull colemanword/geodb:latest
    
GeoDB is a persistant geospatial database built using [Badger](https://github.com/dgraph-io/badger) and gRPC

## Features

- [x] Real-Time Server-Client Object Geolocation Streaming
- [x] Persistent Object Geolocation
- [x] Geolocation Expiration
- [x] Geolocation Boundary Scanning
- [x] Targetted Geofencing- Track objects in relation to others using object "trackers"
- [x] Google Maps Integration(see environmental variables) - Enhance Object Tracking Features 
- [x] gRPC Protocol
- [x] Prometheus Metrics (/metrics endpoint)
- [x] Object Geolocation timeseries exposed with Prometheus metrics
- [x] Configurable(12-factor)
- [x] Basic Authentication
- [x] Docker Image
- [ ] REST Translation Layer
- [x] Docker Compose File
- [ ] Kubernetes Manifests

## Methodology

- Clients may query the database in three ways keys(unique ids), prefix-scanning, or regex 
- Clients can open and execute logic on object geolocation streams that can be filtered by keys(unique ids), prefix-scanning, or regex
- Clients can manage object-centric, dynamic geofences(trackers) that can be used to track an objects location in relation to other registered objects
- Haversine formula is used to calculate whether objects are overlapping using object coordinates and their radius.
- If the server has a google maps api key present in its environmental variables, all geofencing(trackers) will be enhanced with html directions, estimated time of arrival, and more.

## Use Cases
- Ride Sharing
- Food Delivery
- Asset Tracking

## Environmental Variables

- GEODB_PORT (optional) default: :8080
- GEODB_PATH (optional) default: /tmp/geodb
- GEODB_GC_INTERVAL (optional) default: 5m
- GEODB_PASSWORD (optional) 
- GEODB_GMAPS_KEY (optional)

## Sample Docker Compose

```yaml
version: '3.7'
services:
  db:
    image: colemanword/geodb:latest
    env_file:
      - geodb.env
    ports:
      - "8080:8080"
    volumes:
      - default:/tmp/geodb
    networks:
      default:
        aliases:
          - geodb
networks:
  default:

volumes:
  default:

```

## API REF

```proto
syntax = "proto3";

package api;

option go_package = "api";
import "github.com/mwitkow/go-proto-validators/validator.proto";

service GeoDB {
    //Ping - input: empty, output: returns ok if server is healthy.
    rpc Ping(PingRequest) returns(PingResponse){};
    //Set - input: an object output: an object detail. Object details are enhanced when the google maps integration is active
    rpc Set(SetRequest) returns(SetResponse){};
    //Get - input: an array of object keys, output: returns an array of current object details
    rpc Get(GetRequest) returns(GetResponse){};
    //GetRegex - input: a regex string, output: returns an array of current object details with keys that match the regex pattern
    rpc GetRegex(GetRegexRequest) returns(GetRegexResponse){};
    //GetPrefix - input: a prefix string, output: returns an array of current object details with keys that have the given prefix
    rpc GetPrefix(GetPrefixRequest) returns(GetPrefixResponse){};
    //GetKeys -  input: none, output: returns all keys in database
    rpc GetKeys(GetKeysRequest) returns(GetKeysResponse){};
    //GetRegexKeys -  input: a regex string, output: returns all keys in database that match the regex pattern
    rpc GetRegexKeys(GetRegexKeysRequest) returns(GetRegexKeysResponse){};
    //GetPrefixKeys - input: a prefix string, output: returns an array of of keys that have the given prefix
    rpc GetPrefixKeys(GetPrefixKeysRequest) returns(GetPrefixKeysResponse){};
    //Delete -  input: an array of object key strings to delete, output: none
    rpc Delete(DeleteRequest) returns(DeleteResponse){};
    //Stream -  input: a clientID(optional) and an array of object keys(optional),
    //output: a stream of object details for realtime, targetted object geolocation updates
    rpc Stream(StreamRequest) returns(stream StreamResponse){};
    //StreamRegex -  input: a clientID(optional) a regex string,
    //output: a stream of object details for realtime, targetted object geolocation updates that match the regex pattern
    rpc StreamRegex(StreamRegexRequest) returns(stream StreamRegexResponse){};
    //StreamPrefix -  input: a clientID(optional) a prefix string,
    //output: a stream of object details for realtime, targetted object geolocation updates that match the prefix pattern
    rpc StreamPrefix(StreamPrefixRequest) returns(stream StreamPrefixResponse){};

    //ScanBound -  input: a geolocation boundary, output: returns an array of current object details that are within the boundary
    rpc ScanBound(ScanBoundRequest) returns(ScanBoundResponse){};
    //ScanRegexBound -  input: a geolocation boundary, string-array of unique object ids(optional), output: returns an array of current object details that have keys that match the regex and are within the boundary and
    rpc ScanRegexBound(ScanRegexBoundRequest) returns(ScanRegexBoundResponse){};
    //ScanPrefexBound -  input: a geolocation boundary, output: returns an array of current object details that have keys that match the prefix and are within the boundary and
    rpc ScanPrefixBound(ScanPrefixBoundRequest) returns(ScanPrefixBoundResponse){};
    //GetPoint can be used to get an addresses latitude/longitude - google maps integration is required.
    rpc GetPoint(GetPointRequest) returns(GetPointResponse){};
}

//A Point is a simple X/Y or Lng/Lat 2d point. [X, Y] or [Lng, Lat]
message Point {
    double lat =1; //latitude
    double lon =2; //longitude
}

//A Bound represents an enclosed "box" in the 2D Euclidean or Cartesian plane.
message Bound {
    Point corner =1;
    Point opposite_corner =2;
}

//An Object represents anything that has a unique identifier, and a geolocation.
message Object {
    string key = 1 [(validator.field) = {regex: "^.{1,225}$"}]; //a unique identifier
    Point point =2 [(validator.field) = {msg_exists : true}]; //geolocation lat/lon
    int64 radius =3 [(validator.field) = {int_gt: 0}]; //radius of object in meters
    ObjectTracking tracking =4; //ObjectTracking configures object-object geofencing, directions, eta, etc
    map<string, string> metadata =5; //optional metadata associated with the object
    bool get_address =6;
    bool get_timezone =7;
    int64 expires_unix =8; //a unix timestamp in the future when the database should clean up the object. empty if no expiration.
    int64 updated_unix =9; //unix timestamp representing last update (optional)
}

//ObjectTracking configures object-object geofencing, directions, eta, etc
message ObjectTracking {
    TravelMode travel_mode =1; //defaults to driving
    repeated ObjectTracker trackers =2; //an array of foreigm object keys that represent other objects you want to track the distance, eta, directions, etc(see tracker)
}

//a foreign object to track against another object
message ObjectTracker {
    string target_object_key =1 [(validator.field) = {regex: "^.{1,225}$"}];
    bool track_directions =2;
    bool track_distance =3;
    bool track_eta =4;
}

//Directions if using the google maps integration
message Directions  {
    string html_directions =1;
    int64 eta =2;
    int64 travel_dist =3;
}

//A human readable address that is generated from a lat,lon if using the google maps integration
message Address {
    string state =1;
    string address =2;
    string country =3;
    string zip =4;
    string county =5;
    string city =6;
}

//Tracker is data associated with the object tracking mechanism- it tracks one obects relation to another.
//An object can have many trackers representing a one-many relationship
message TrackerEvent {
    Object object =1; //targe object
    double distance =2; //distance to object
    bool inside =3; //whether objects are overlapping
    Directions direction =4; //directions from one object to another (base64 encoded)
    int64 timestamp_unix =5;
}

//ObjectDetail is an enhanced view of an Object containing a human readable address and the objects latest tracking information
message ObjectDetail {
    Object object =1;
    Address address = 2;
    string timezone =3;
    repeated TrackerEvent events =4;
}

//TravelMode is used to generate directions based on the type of travel the object is utilizing. only necessary if using google maps
enum TravelMode {
    Driving = 0;
    Walking =1;
    Bicycling =2;
    Transit =3;
}

message StreamRequest {
    string client_id =1;
    repeated string keys =2;
}

message StreamResponse {
    ObjectDetail object =1;
}

message StreamRegexRequest {
    string client_id =1;
    string regex =2 [(validator.field) = {regex: "^.{1,225}$"}];
}

message StreamRegexResponse {
    ObjectDetail object =1;
}

message StreamPrefixRequest {
    string client_id =1;
    string prefix =2 [(validator.field) = {regex: "^.{1,225}$"}];
}

message StreamPrefixResponse {
    ObjectDetail object =1;
}

message SetRequest {
    Object object =1 [(validator.field) = {msg_exists : true}];
}

message SetResponse {
    ObjectDetail object= 1;
}

message GetKeysRequest {}

message GetKeysResponse {
    repeated string keys =1;
}

message GetPrefixKeysRequest {
    string prefix =1 [(validator.field) = {regex: "^.{1,225}$"}];
}

message GetPrefixKeysResponse {
    repeated string keys =1;
}

message GetRegexKeysRequest {
    string regex =1 [(validator.field) = {regex: "^.{1,225}$"}];
}

message GetRegexKeysResponse {
    repeated string keys =1;
}

message GetRequest {
    repeated string keys =1;
}

message GetResponse {
    map<string, ObjectDetail> objects= 1;
}

message GetRegexRequest {
    string regex =1 [(validator.field) = {regex: "^.{1,225}$"}];
}

message GetRegexResponse {
    map<string, ObjectDetail> objects= 1;
}

message GetPrefixRequest {
    string prefix =1 [(validator.field) = {regex: "^.{1,225}$"}];
}

message GetPrefixResponse {
    map<string, ObjectDetail> objects= 1;
}

message DeleteRequest {
    repeated string keys =1;
}

message DeleteResponse {}

message ScanBoundRequest {
    Bound bound =1;
    repeated string keys =2; //if zero keys present, ScanBound will scan the entire database
}

message ScanBoundResponse {
    map<string, ObjectDetail> objects= 1;
}

message ScanPrefixBoundRequest {
    Bound bound =1;
    string prefix =2;
}

message ScanPrefixBoundResponse {
    map<string, ObjectDetail> objects= 1;
}

message ScanRegexBoundRequest {
    Bound bound =1;
    string regex =2;
}

message ScanRegexBoundResponse {
    map<string, ObjectDetail> objects= 1;
}

message GetPointRequest {
    string address =1;
}

message GetPointResponse {
    Point point =1;
}

message PingRequest {}

message PingResponse {
    bool ok =1;
}
```
