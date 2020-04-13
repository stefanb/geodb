package services

import (
	"context"
	api "github.com/autom8ter/geodb/gen/go/geodb"
	"github.com/autom8ter/geodb/maps"
	"github.com/autom8ter/geodb/stream"
	"github.com/dgraph-io/badger/v2"
)

type GeoDB struct {
	hub   *stream.Hub
	db    *badger.DB
	gmaps *maps.Client
}

func NewGeoDB(db *badger.DB, hub *stream.Hub, gmaps *maps.Client) *GeoDB {
	return &GeoDB{
		hub:   hub,
		db:    db,
		gmaps: gmaps,
	}
}

func (p *GeoDB) Ping(ctx context.Context, req *api.PingRequest) (*api.PingResponse, error) {
	return &api.PingResponse{
		Ok: true,
	}, nil
}
