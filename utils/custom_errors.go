package custom_errors

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ERR_MISSING_METADATA = status.Error(codes.InvalidArgument, "missing metadata")
var ERR_INVALID_TOKEN = status.Error(codes.Unauthenticated, "invalid token")
