package interceptors

import (
	"context"

	"github.com/JustUzair/go-grpc-irctc-backend/utils/auth"
	"github.com/JustUzair/go-grpc-irctc-backend/utils/env"
	grpcInterceptors "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	grpcAuth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	grpcSelector "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/selector"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func AccessTokenAuth(config env.Config) grpcAuth.AuthFunc {
	return func(ctx context.Context) (context.Context, error) {
		token, err := grpcAuth.AuthFromMD(ctx, "bearer")
		if err != nil {
			return nil, err
		}
		claims, err := auth.VerifyAccessToken(token, config)
		if err != nil {
			return nil, status.Error(
				codes.Unauthenticated,
				"invalid access token",
			)
		}
		return auth.WithUserID(ctx, claims.UserID), nil
	}
}

func MatchMethods(methods ...string) func(context.Context, grpcInterceptors.CallMeta) bool {
	protected := make(map[string]struct{}, len(methods))
	for _, method := range methods {
		protected[method] = struct{}{}
	}
	return func(ctx context.Context, call grpcInterceptors.CallMeta) bool {
		_, ok := protected[call.FullMethod()]
		return ok
	}
}

func UnaryServerAuthInterceptor(
	config env.Config,
	protectedMethods ...string,
) grpc.UnaryServerInterceptor {
	return grpcSelector.UnaryServerInterceptor(
		grpcAuth.UnaryServerInterceptor(AccessTokenAuth(config)),
		grpcSelector.MatchFunc(MatchMethods(protectedMethods...)),
	)
}
