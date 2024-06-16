package interceptors

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type AuthClientInterceptor struct {
	accessToken string
}

func NewAuthClientInterceptor() *AuthClientInterceptor {
	return &AuthClientInterceptor{
		accessToken: "",
	}
}

func (i *AuthClientInterceptor) Unary() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		slog.With("method", method).Debug("--> unary auth client interceptor triggered")
		return invoker(i.attachToken(ctx), method, req, reply, cc, opts...)
	}
}

func (i *AuthClientInterceptor) Stream() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		slog.With("method", method).Debug("--> stream auth client interceptor triggered")
		i.accessToken = "Bearer token"
		return streamer(i.attachToken(ctx), desc, cc, method, opts...)
	}
}

func (i *AuthClientInterceptor) attachToken(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", i.accessToken)
}
