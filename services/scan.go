package services

import (
	"context"
	"github.com/autom8ter/geodb/db"
	api "github.com/autom8ter/geodb/gen/go/geodb"
)

func (p *GeoDB) ScanBound(ctx context.Context, r *api.ScanBoundRequest) (*api.ScanBoundResponse, error) {
	objects, err := db.ScanBound(p.db, r.Bound, r.Keys)
	if err != nil {
		return nil, err
	}
	return &api.ScanBoundResponse{
		Objects: objects,
	}, nil
}

func (p *GeoDB) ScanRegexBound(ctx context.Context, r *api.ScanRegexBoundRequest) (*api.ScanRegexBoundResponse, error) {
	objects, err := db.ScanRegexBound(p.db, r.Bound, r.Regex)
	if err != nil {
		return nil, err
	}
	return &api.ScanRegexBoundResponse{
		Objects: objects,
	}, nil
}

func (p *GeoDB) ScanPrefixBound(ctx context.Context, r *api.ScanPrefixBoundRequest) (*api.ScanPrefixBoundResponse, error) {
	objects, err := db.ScanPrefixBound(p.db, r.Bound, r.Prefix)
	if err != nil {
		return nil, err
	}
	return &api.ScanPrefixBoundResponse{
		Objects: objects,
	}, nil
}
