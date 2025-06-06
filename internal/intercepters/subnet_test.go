package intercepters

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestSubnetIPInterceptor(t *testing.T) {
	// A dummy handler that returns the real-ip value from context if present
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		ip, _ := ctx.Value(RealIPKey).(string)
		return ip, nil
	}

	tests := []struct {
		name   string
		ctx    context.Context
		wantIP string
	}{
		{
			name:   "with x-real-ip metadata",
			ctx:    metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-real-ip", "192.168.1.100")),
			wantIP: "192.168.1.100",
		},
		{
			name:   "with empty x-real-ip metadata",
			ctx:    metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-real-ip", "")),
			wantIP: "",
		},
		{
			name:   "without metadata",
			ctx:    context.Background(),
			wantIP: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := SubnetIPInterceptor(tt.ctx, nil, &grpc.UnaryServerInfo{
				FullMethod: "/test.TestMethod",
			}, handler)
			if err != nil {
				t.Fatalf("Interceptor returned error: %v", err)
			}
			gotIP, _ := resp.(string)
			if gotIP != tt.wantIP {
				t.Errorf("got IP = %q, want %q", gotIP, tt.wantIP)
			}
		})
	}
}
