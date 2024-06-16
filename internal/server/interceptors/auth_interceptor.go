package interceptors

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthServerInterceptor struct {
	accessibleRoles []string
}

func NewAuthServerInterceptor(accessibleRoles []string) *AuthServerInterceptor {
	return &AuthServerInterceptor{accessibleRoles}
}

func (interceptor *AuthServerInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		slog.With("method", info.FullMethod).Debug("--> unary auth server interceptor")

		err := interceptor.authorize(ctx)
		if err != nil {
			slog.Error("Unauthorized: ", err)
			return nil, err
		}

		return handler(ctx, req)
	}
}

func (interceptor *AuthServerInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		slog.With("method", info.FullMethod).Debug("--> stream auth server interceptor triggered")
		// TODO: implement streaming authorization
		return handler(srv, stream)
	}
}

func (interceptor *AuthServerInterceptor) authorize(ctx context.Context) error {
	accessibleRoles := interceptor.accessibleRoles
	if len(accessibleRoles) == 0 {
		// everyone can access
		return nil
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "metadata is not provided")
	}

	values := md["authorization"]
	if len(values) == 0 {
		return status.Errorf(codes.Unauthenticated, "authorization token is not provided")
	}

	accessToken := values[0]
	if isValid := validateToken(accessToken); isValid {
		return nil
	}

	return status.Error(codes.PermissionDenied, "no permission to access this RPC")
}

func validateToken(_ string) bool {
	return true
}
