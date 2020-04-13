package services

import (
	"context"
	api "github.com/autom8ter/geodb/gen/go/geodb"
)

func (p *GeoDB) ScanBound(context.Context, *api.ScanBoundRequest) (*api.ScanBoundResponse, error) {
	panic("implement me")
}

func (p *GeoDB) ScanRegexBound(context.Context, *api.ScanRegexBoundRequest) (*api.ScanRegexBoundResponse, error) {
	panic("implement me")
}

func (p *GeoDB) ScanPrefexBound(context.Context, *api.ScanPrefixBoundRequest) (*api.ScanPrefixBoundResponse, error) {
	panic("implement me")
}
