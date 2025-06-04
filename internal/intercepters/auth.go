package intercepters

import (
	"context"
	"strings"

	"github.com/atinyakov/go-url-shortener/internal/app/service"
	"github.com/atinyakov/go-url-shortener/internal/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func WithJWT(auth service.AuthIface) func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		var userID string

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeader := md.Get("authorization")

		// If there's no token, generate a new one
		if len(authHeader) == 0 {
			token, generatedID, err := auth.BuildJWTString()
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to build JWT: %v", err)
			}
			// Client must handle receiving the new token (e.g., as trailer or custom field)
			userID = generatedID

			// Optionally, return it in the trailer metadata
			grpc.SetTrailer(ctx, metadata.Pairs("new-token", token))
		} else {
			// "Bearer <token>"
			tokenString := strings.TrimPrefix(authHeader[0], "Bearer ")
			claims, err := auth.ParseRawJWT(tokenString)
			if err != nil {
				return nil, status.Errorf(codes.Unauthenticated, "invalid JWT: %v", err)
			}
			userID = claims.UserID
		}

		// Inject userID into context
		ctx = context.WithValue(ctx, middleware.UserIDKey, userID)

		return handler(ctx, req)
	}
}
