package interceptors

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type contextKey string

const metaContextKey contextKey = "user_meta"

type UserMeta struct {
	UserAgent         string
	IP                string
	Accept            string
	DeviceFingerprint string
}

func MetaInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	meta := &UserMeta{}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		meta.UserAgent = getFirst(md.Get("user-agent"))
		meta.Accept = getFirst(md.Get("accept"))
		// IP is usually passed by API Gateway via x-forwarded-for or x-real-ip
		ip := getFirst(md.Get("x-forwarded-for"))
		if ip == "" {
			ip = getFirst(md.Get("x-real-ip"))
		}
		meta.IP = ip
	}
	// Calculate fingerprint inside the interceptor centrally!
	meta.DeviceFingerprint = GetDeviceFingerprint(
		meta.UserAgent,
		meta.IP,
		meta.Accept,
	)
	newCtx := context.WithValue(ctx, metaContextKey, meta)
	return handler(newCtx, req)
}

// Helper to get first value from metadata slice
func getFirst(vals []string) string {
	if len(vals) > 0 {
		// x-forwarded-for can be a comma-separated list: "client, proxy1, proxy2"
		return strings.TrimSpace(strings.Split(vals[0], ",")[0])
	}
	return ""
}

// Helper function for handlers to easily retrieve Meta from context
func GetMetaFromContext(ctx context.Context) (*UserMeta, bool) {
	meta, ok := ctx.Value(metaContextKey).(*UserMeta)
	return meta, ok
}

// User Device Fingerprint
// GetDeviceFingerprint calculates a SHA-256 hash from user agent, IP address, and accept header
func GetDeviceFingerprint(userAgent, ip, accept string) string {
	raw := fmt.Sprintf("%s|%s|%s", userAgent, ip, accept)
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}
