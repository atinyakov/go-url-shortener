package intercepters

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type contextKey string

const RealIPKey contextKey = "real-ip"

func SubnetIPInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if ips := md.Get("x-real-ip"); len(ips) > 0 {
			ctx = context.WithValue(ctx, RealIPKey, ips[0])
		}
	}
	return handler(ctx, req)
}
