package services

import (
	"context"
	"github.com/autom8ter/geodb/db"
	api "github.com/autom8ter/geodb/gen/go/geodb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (p *GeoDB) Set(ctx context.Context, r *api.SetRequest) (*api.SetResponse, error) {
	if err := r.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	objects, err := db.Set(p.db, p.gmaps, p.hub, r.Objects)
	if err != nil {
		return nil, err
	}
	return &api.SetResponse{
		Objects: objects,
	}, nil
}

func (p *GeoDB) GetRegex(ctx context.Context, r *api.GetRegexRequest) (*api.GetRegexResponse, error) {
	objects, err := db.GetRegex(p.db, r.Regex)
	if err != nil {
		return nil, err
	}
	return &api.GetRegexResponse{
		Objects: objects,
	}, nil
}

func (p *GeoDB) Get(ctx context.Context, r *api.GetRequest) (*api.GetResponse, error) {
	objects, err := db.Get(p.db, r.Keys)
	if err != nil {
		return nil, err
	}
	return &api.GetResponse{
		Objects: objects,
	}, nil
}

func (p *GeoDB) GetPrefix(ctx context.Context, r *api.GetPrefixRequest) (*api.GetPrefixResponse, error) {
	objects, err := db.GetPrefix(p.db, r.Prefix)
	if err != nil {
		return nil, err
	}
	return &api.GetPrefixResponse{
		Objects: objects,
	}, nil
}

func (p *GeoDB) Delete(ctx context.Context, r *api.DeleteRequest) (*api.DeleteResponse, error) {
	if err := db.Delete(p.db, r.Keys); err != nil {
		return nil, err
	}
	return &api.DeleteResponse{}, nil
}
