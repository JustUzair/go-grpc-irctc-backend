package custom_errors

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ERR_MISSING_METADATA = status.Error(codes.InvalidArgument, "missing metadata")
var ERR_INVALID_TOKEN = status.Error(codes.Unauthenticated, "invalid token")
var ERR_INVALID_CONFIG = status.Error(codes.Internal, "invalid configuration")
var ERR_INVALID_TEMPLATE = status.Error(codes.NotFound, "invalid email template")
var ERR_BAD_REQUEST = status.Error(codes.InvalidArgument, "bad request")
var ERR_PASSWORD_MISMATCH = status.Error(codes.InvalidArgument, "passwords do not match")
var ERR_TOO_MANY_REQUESTS = status.Error(codes.ResourceExhausted, "too many requests")
