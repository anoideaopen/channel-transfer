package middleware

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthenticationHeader declares the standard header that should contain
// authentication data such as a token, key, etc.
const AuthenticationHeader = "authorization"

// NewGRPCAuthenticationInterceptor creates an interceptor for GRPC requests. It
// checks for the presence of [AuthenticationHeader] and verifies that the token
// matches the parameter set in the constructor.
func NewGRPCAuthenticationInterceptor(
	token string,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo, //nolint:revive
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(
				codes.Unauthenticated,
				"metadata not provided",
			)
		}

		if len(md.Get(AuthenticationHeader)) != 1 {
			return nil, status.Error(
				codes.Unauthenticated,
				"auth header not provided",
			)
		}

		if md.Get(AuthenticationHeader)[0] != token {
			return nil, status.Error(
				codes.Unauthenticated,
				"invalid auth header",
			)
		}

		return handler(ctx, req)
	}
}
