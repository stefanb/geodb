# GeoDB- A Persistent Geospatial Database

    go get github.com/autom8ter/geodb
    
GeoDB is a persistant geospatial database built using [Badger](https://github.com/dgraph-io/badger) and gRPC

## Features

- [x] Real-Time server-client object geolocation streaming
- [x] Persistent object geolocation
- [x] Expire object geolocation streaming
- [x] Track objects in relation to others using object "trackers"
- [x] Enhance object tracking features using Google Maps Integration(see environmental variables)
- [x] gRPC protocol
- [x] Logging
- [x] Prometheus metrics (/metrics endpoint)
- [x] Configurable(12-factor)
- [x] Basic Authentication

## Environmental Variables

- GEODB_PORT (optional) default: :8080
- GEODB_PATH (optional) default: /tmp/geodb
- GEODB_GC_INTERVAL (optional) default: 5m
- GEODB_PASSWORD (optional) 
- GEODB_GMAPS_KEY (optional)

## API REF

```proto
syntax = "proto3";

package api;

option go_package = "api";
import "github.com/mwitkow/go-proto-validators/validator.proto";

service GeoDB {
    //Ping - input: empty, output: returns ok if server is healthy.
    rpc Ping(PingRequest) returns(PingResponse){};
    //Set - input: a string(objectKey) Object map output: returns an array of updated object details.
    //objects are upserted in the order they are sent
    rpc Set(SetRequest) returns(SetResponse){};
    //Get - input: an array of object keys, output: returns an array of current object details
    rpc Get(GetRequest) returns(GetResponse){};
    //GetRegex - input: a regex string, output: returns an array of current object details with keys that match the regex pattern
    rpc GetRegex(GetRegexRequest) returns(GetRegexResponse){};
    //Seek - input: a prefix string, output: returns an array of current object details with keys that have the given prefix
    rpc Seek(SeekRequest) returns(SeekResponse){};
    //SeekKeys - input: a prefix string, output: returns an array of of keys that have the given prefix
    rpc SeekKeys(SeekKeysRequest) returns(SeekKeysResponse){};
    //GetKeys -  input: none, output: returns all keys in database
    rpc GetKeys(GetKeysRequest) returns(GetKeysResponse){};
    //GetRegexKeys -  input: a regex string, output: returns all keys in database that match the regex pattern
    rpc GetRegexKeys(GetRegexKeysRequest) returns(GetRegexKeysResponse){};
    //Delete -  input: an array of object key strings to delete, output: none
    rpc Delete(DeleteRequest) returns(DeleteResponse){};
    //Stream -  input: a clientID(optional) and an array of object keys(optional),
    //output: a stream of object details for realtime, targetted object geolocation updates
    rpc Stream(StreamRequest) returns(stream StreamResponse){};
    //StreamRegex -  input: a clientID(optional) a regex string(optional),
    //output: a stream of object details for realtime, targetted object geolocation updates that match the regex pattern
    rpc StreamRegex(StreamRegexRequest) returns(stream StreamRegexResponse){};
}

message Point {
    double lat =1;
    double lon =2;
}

//An Object represents anything that has a unique identifier, and a geolocation.
message Object {
    string key =1; //a unique identifier
    Point point =2; //geolocation lat/lon
    int64 radius =3; //defaults to 100(meters)
    TravelMode travel_mode =4; //defaults to driving
    map<string, string> metadata =5; //optional metadata associated with the object
    int64 expires_unix =6; //a unix timestamp in the future when the database should clean up the object. empty if no expiration.
    repeated string trackers =7; //an array of foreigm object keys that represent other objects you want to track the distance, eta, directions, etc(see tracker)
    int64 updated_unix =8; //unix timestamp representing last update (optional)
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
message Tracker {
    Object object =1; //targe object
    double distance =2; //distance to object
    bool inside =3; //whether objects are overlapping
    Directions direction =4; //directions from one object to annother
    int64 timestamp_unix =5;
}

//ObjectDetail is an enhanced view of an Object containing a human readable address and the objects latest tracking information
message ObjectDetail {
    Object object =1;
    Address address = 2;
    repeated Tracker trackers =3;
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

message SetRequest {
    map<string, Object> object =1;
}

message SetResponse {
    map<string, ObjectDetail> object= 1;
}

message GetKeysRequest {}

message GetKeysResponse {
    repeated string keys =1;
}

message SeekKeysRequest {
    string prefix =1 [(validator.field) = {regex: "^.{1,225}$"}];
}

message SeekKeysResponse {
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
    map<string, ObjectDetail> object= 1;
}

message GetRegexRequest {
    string regex =1 [(validator.field) = {regex: "^.{1,225}$"}];
}

message GetRegexResponse {
    map<string, ObjectDetail> object= 1;
}

message SeekRequest {
    string prefix =1 [(validator.field) = {regex: "^.{1,225}$"}];
}

message SeekResponse {
    map<string, ObjectDetail> object= 1;
}

message DeleteRequest {
    repeated string keys =1;
}

message DeleteResponse {}

message PingRequest {}

message PingResponse {
    bool ok =1;
}
```
