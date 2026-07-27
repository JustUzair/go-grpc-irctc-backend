package logger

import (
	"context"
	"log"
	"strings"

	custom_errors "github.com/JustUzair/go-grpc-irctc-backend/utils/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func UnaryServerAuthInterceptor(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, custom_errors.ERR_MISSING_METADATA
	}

	if !valid(md["authorization"]) {
		return nil, custom_errors.ERR_INVALID_TOKEN
	}
	m, err := handler(ctx, req)
	if err != nil {
		log.Default().Fatalf("RPC failed with error: %v\n", err)
	}
	return m, err

}

// valid validates the authorization.
func valid(authorization []string) bool {
	if len(authorization) < 1 {
		return false
	}
	token := strings.TrimPrefix(authorization[0], "Bearer ")
	// Perform the token validation here. For the sake of this example, the code
	// here forgoes any of the usual OAuth2 token validation and instead checks
	// for a token matching an arbitrary string.
	return token == "some-secret-token"
}
