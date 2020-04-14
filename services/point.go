package services

import (
	"context"
	api "github.com/autom8ter/geodb/gen/go/geodb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (p *GeoDB) GetPoint(ctx context.Context, r *api.GetPointRequest) (*api.GetPointResponse, error) {
	if p.gmaps != nil {
		point, err := p.gmaps.GetCoordinates(r.Address)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return &api.GetPointResponse{
			Point: point,
		}, nil
	}
	return nil, status.Error(codes.Unimplemented, "google maps integration not set up")
}
