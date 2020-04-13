package services

import (
	"context"
	"github.com/autom8ter/geodb/db"
	api "github.com/autom8ter/geodb/gen/go/geodb"
)

func (p *GeoDB) GetKeys(ctx context.Context, r *api.GetKeysRequest) (*api.GetKeysResponse, error) {
	return &api.GetKeysResponse{
		Keys: db.GetKeys(p.db),
	}, nil
}

func (p *GeoDB) GetPrefixKeys(ctx context.Context, r *api.GetPrefixKeysRequest) (*api.GetPrefixKeysResponse, error) {
	return &api.GetPrefixKeysResponse{
		Keys: db.GetPrefixKeys(p.db, r.Prefix),
	}, nil
}

func (p *GeoDB) GetRegexKeys(ctx context.Context, r *api.GetRegexKeysRequest) (*api.GetRegexKeysResponse, error) {
	keys, err := db.GetRegexKeys(p.db, r.Regex)
	if err != nil {
		return nil, err
	}
	return &api.GetRegexKeysResponse{
		Keys: keys,
	}, nil
}
