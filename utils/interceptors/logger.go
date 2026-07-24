package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"

	grpcLogging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/lmittmann/tint"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var serviceLogger = newServiceLogger()

var unaryServerLogger = grpcLogging.UnaryServerInterceptor(
	interceptorLogger(serviceLogger),
	grpcLogging.WithLogOnEvents(grpcLogging.FinishCall),
	grpcLogging.WithDurationField(grpcLogging.DurationToDurationField),
	grpcLogging.WithFieldsFromContext(requestFieldsFromContext),
)

func UnaryServerLoggerInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	return unaryServerLogger(ctx, req, info, handler)
}

func interceptorLogger(logger *slog.Logger) grpcLogging.Logger {
	return grpcLogging.LoggerFunc(
		func(ctx context.Context, level grpcLogging.Level, message string, fields ...any) {
			logger.Log(ctx, slog.Level(level), message, fields...)
		},
	)
}

func newServiceLogger() *slog.Logger {
	level := configuredLogLevel()

	if strings.EqualFold(strings.TrimSpace(os.Getenv("LOG_FORMAT")), "json") {
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		}))
	}

	handler := tint.NewTextHandler(os.Stdout, &tint.Options{
		Level:       level,
		TimeFormat:  "15:04:05.000",
		NoColor:     os.Getenv("NO_COLOR") != "",
		ReplaceAttr: compactConsoleAttribute,
	})

	return slog.New(handler)
}

func compactConsoleAttribute(_ []string, attribute slog.Attr) slog.Attr {
	switch attribute.Key {
	case "protocol", "grpc.component", "grpc.method_type", "peer.address", "grpc.start_time":
		return slog.Attr{}
	case "grpc.service":
		attribute.Key = "rpc.service"
	case "grpc.method":
		attribute.Key = "rpc.method"
	case "grpc.code":
		attribute.Key = "rpc.code"
	case "grpc.duration":
		attribute.Key = "duration"
	}

	return attribute
}

func configuredLogLevel() slog.Level {
	value := strings.TrimSpace(os.Getenv("LOG_LEVEL"))
	if value == "" {
		return slog.LevelInfo
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(value)); err != nil {
		return slog.LevelInfo
	}

	return level
}

func requestFieldsFromContext(ctx context.Context) grpcLogging.Fields {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil
	}

	for _, key := range []string{"x-request-id", "request-id"} {
		if values := md.Get(key); len(values) > 0 {
			if requestID := strings.TrimSpace(values[0]); requestID != "" {
				return grpcLogging.Fields{"request_id", requestID}
			}
		}
	}

	return nil
}
