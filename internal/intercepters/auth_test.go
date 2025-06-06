package intercepters_test

import (
	"context"
	"strings"
	"testing"

	"github.com/atinyakov/go-url-shortener/internal/app/service"
	"github.com/atinyakov/go-url-shortener/internal/intercepters"
	"github.com/atinyakov/go-url-shortener/internal/middleware"
	"github.com/atinyakov/go-url-shortener/internal/mocks"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestWithJWT(t *testing.T) {
	const (
		expectedUserID   = "user-123"
		expectedToken    = "token-abc"
		expectedNewToken = "new-token-xyz"
	)

	tests := []struct {
		name              string
		md                metadata.MD
		buildJWTStringRes struct {
			token  string
			userID string
			err    error
		}
		parseRawJWTRes struct {
			claims *service.Claims
			err    error
		}
		wantErrCode   codes.Code
		wantUserID    string
		expectTrailer bool
	}{
		{
			name:        "missing metadata",
			md:          nil,
			wantErrCode: codes.Unauthenticated,
		},
		{
			name: "no token in metadata calls BuildJWTString",
			md:   metadata.Pairs(), // no "authorization"
			buildJWTStringRes: struct {
				token  string
				userID string
				err    error
			}{
				token:  expectedNewToken,
				userID: expectedUserID,
				err:    nil,
			},
			wantUserID:    expectedUserID,
			expectTrailer: true,
		},
		{
			name: "BuildJWTString returns error",
			md:   metadata.Pairs(), // no "authorization"
			buildJWTStringRes: struct {
				token  string
				userID string
				err    error
			}{
				token:  "",
				userID: "",
				err:    context.Canceled, // example error
			},
			wantErrCode: codes.Internal,
		},
		{
			name: "invalid token returns Unauthenticated",
			md:   metadata.Pairs("authorization", "Bearer invalidtoken"),
			parseRawJWTRes: struct {
				claims *service.Claims
				err    error
			}{
				claims: nil,
				err:    context.Canceled, // example parse error
			},
			wantErrCode: codes.Unauthenticated,
		},
		{
			name: "valid token parses userID and calls handler",
			md:   metadata.Pairs("authorization", "Bearer "+expectedToken),
			parseRawJWTRes: struct {
				claims *service.Claims
				err    error
			}{
				claims: &service.Claims{UserID: expectedUserID},
				err:    nil,
			},
			wantUserID: expectedUserID,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockAuth := mocks.NewMockAuthIface(ctrl)
			interceptor := intercepters.WithJWT(mockAuth)

			if tt.md == nil {
				// No metadata at all → do NOT expect BuildJWTString()
				// Just skip expectation, interceptor should return unauthenticated before calling mock
			} else if len(tt.md["authorization"]) == 0 {
				// Metadata present but no "authorization" header → expect BuildJWTString()
				mockAuth.EXPECT().
					BuildJWTString().
					Return(tt.buildJWTStringRes.token, tt.buildJWTStringRes.userID, tt.buildJWTStringRes.err).
					Times(1)
			} else {
				// Token present → expect ParseRawJWT
				authHeader := tt.md["authorization"]
				var token string
				if len(authHeader) > 0 {
					token = strings.TrimPrefix(authHeader[0], "Bearer ")
				}
				mockAuth.EXPECT().
					ParseRawJWT(token).
					Return(tt.parseRawJWTRes.claims, tt.parseRawJWTRes.err).
					Times(1)
			}

			// Dummy handler returns fixed string and nil error
			handler := func(ctx context.Context, req interface{}) (interface{}, error) {
				// Check userID injected in context if expected
				if tt.wantUserID != "" {
					gotUserID := ctx.Value(middleware.UserIDKey)
					if gotUserID != tt.wantUserID {
						t.Errorf("userID in context = %v, want %v", gotUserID, tt.wantUserID)
					}
				}
				return "ok", nil
			}

			ctx := context.Background()
			if tt.md != nil {
				ctx = metadata.NewIncomingContext(ctx, tt.md)
			}

			resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{
				FullMethod: "/test/method",
			}, handler)

			if tt.wantErrCode != codes.OK && tt.wantErrCode != 0 {
				if err == nil {
					t.Fatalf("expected error code %v but got no error", tt.wantErrCode)
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected grpc status error but got %v", err)
				}
				if st.Code() != tt.wantErrCode {
					t.Fatalf("expected error code %v but got %v", tt.wantErrCode, st.Code())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp != "ok" {
				t.Errorf("unexpected response: %v", resp)
			}
		})
	}
}
