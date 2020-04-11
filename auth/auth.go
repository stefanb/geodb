package auth

import (
	"context"
	"github.com/autom8ter/geodb/config"
	"google.golang.org/grpc/codes"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"

	"google.golang.org/grpc/status"
)

func BasicAuthFunc() grpc_auth.AuthFunc {
	return func(ctx context.Context) (context.Context, error) {
		if config.Config.IsSet("GEODB_PASSWORD") {
			basicAuth, err := grpc_auth.AuthFromMD(ctx, "basic")
			if err != nil {
				return nil, status.Errorf(codes.Unauthenticated, "failed to find authentication header with basic scheme\n%v", err)
			}
			if basicAuth != config.Config.GetString("GEODB_PASSWORD") {
				return nil, status.Error(codes.Unauthenticated, "invalid password")
			}
		}
		return ctx, nil
	}
}
