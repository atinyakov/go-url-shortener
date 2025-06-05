package grpc_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/atinyakov/go-url-shortener/internal/app/server/grpc"
	"github.com/atinyakov/go-url-shortener/internal/middleware"
	"github.com/atinyakov/go-url-shortener/internal/mocks"
	"github.com/atinyakov/go-url-shortener/internal/models"
	"github.com/atinyakov/go-url-shortener/internal/repository"
	"github.com/atinyakov/go-url-shortener/internal/storage"
	pb "github.com/atinyakov/go-url-shortener/proto"
)

func TestCreateURLRecord(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockURLService := mocks.NewMockURLServiceIface(ctrl)

	handler := &grpc.ShortenerServer{
		Service: mockURLService,
		BaseURL: "http://localhost",
	}

	ctx := context.WithValue(context.Background(), middleware.UserIDKey, "user123")
	req := &pb.CreateURLRecordRequest{Url: "https://example.com"}

	expected := &storage.URLRecord{
		Short:    "abc123",
		Original: "https://example.com",
		UserID:   "user123",
	}
	mockURLService.EXPECT().
		CreateURLRecord(ctx, req.Url, "user123").
		Return(expected, nil)

	resp, err := handler.CreateURLRecord(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, "http://localhost/abc123", resp.Result)
}

func TestCreateURLRecord_NoUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockURLService := mocks.NewMockURLServiceIface(ctrl)

	handler := &grpc.ShortenerServer{
		Service: mockURLService,
		BaseURL: "http://localhost",
	}

	ctx := context.Background() // no user ID
	_, err := handler.CreateURLRecord(ctx, &pb.CreateURLRecordRequest{Url: "https://example.com"})

	assert.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "user ID missing")
}

func TestGetStats_SubnetDenied(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockURLService := mocks.NewMockURLServiceIface(ctrl)

	handler := &grpc.ShortenerServer{
		Service:       mockURLService,
		TrustedSubnet: "10.0.0.0/24",
	}

	ctx := context.Background() // no RealIPKey in context
	_, err := handler.GetStats(ctx, &emptypb.Empty{})

	assert.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	assert.Contains(t, err.Error(), "X-Real-IP header missing")
}

func TestCreateURLRecords_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockURLService := mocks.NewMockURLServiceIface(ctrl)

	handler := &grpc.ShortenerServer{
		Service: mockURLService,
		BaseURL: "http://localhost",
	}

	ctx := context.WithValue(context.Background(), middleware.UserIDKey, "user123")

	req := &pb.CreateURLRecordBatchRequest{
		Items: []*pb.CreateURLRecordBatchRequestItem{
			{CorrelationId: "c1", OriginalUrl: "https://example1.com"},
			{CorrelationId: "c2", OriginalUrl: "https://example2.com"},
		},
	}

	expectedInput := []models.BatchRequest{
		{CorrelationID: "c1", OriginalURL: "https://example1.com"},
		{CorrelationID: "c2", OriginalURL: "https://example2.com"},
	}

	expectedOutput := &[]models.BatchResponse{
		{CorrelationID: "c1", ShortURL: "http://localhost/a1"},
		{CorrelationID: "c2", ShortURL: "http://localhost/a2"},
	}

	mockURLService.EXPECT().
		CreateURLRecords(gomock.Any(), expectedInput, "user123").
		Return(expectedOutput, nil)

	resp, err := handler.CreateURLRecords(ctx, req)
	assert.NoError(t, err)
	assert.Len(t, resp.Items, 2)
	assert.Equal(t, "c1", resp.Items[0].CorrelationId)
	assert.Equal(t, "http://localhost/a1", resp.Items[0].ShortUrl)
	assert.Equal(t, "c2", resp.Items[1].CorrelationId)
	assert.Equal(t, "http://localhost/a2", resp.Items[1].ShortUrl)
}

func TestCreateURLRecords_NoUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockURLService := mocks.NewMockURLServiceIface(ctrl)

	handler := &grpc.ShortenerServer{
		Service: mockURLService,
		BaseURL: "http://localhost",
	}

	ctx := context.Background() // no user ID

	req := &pb.CreateURLRecordBatchRequest{
		Items: []*pb.CreateURLRecordBatchRequestItem{
			{CorrelationId: "c1", OriginalUrl: "https://example1.com"},
		},
	}

	resp, err := handler.CreateURLRecords(ctx, req)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "user ID missing")
}

func TestCreateURLRecords_Conflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockURLService := mocks.NewMockURLServiceIface(ctrl)

	handler := &grpc.ShortenerServer{
		Service: mockURLService,
		BaseURL: "http://localhost",
	}

	ctx := context.WithValue(context.Background(), middleware.UserIDKey, "user123")

	req := &pb.CreateURLRecordBatchRequest{
		Items: []*pb.CreateURLRecordBatchRequestItem{
			{CorrelationId: "c1", OriginalUrl: "https://example.com"},
		},
	}

	expectedInput := []models.BatchRequest{
		{CorrelationID: "c1", OriginalURL: "https://example.com"},
	}

	mockURLService.EXPECT().
		CreateURLRecords(gomock.Any(), expectedInput, "user123").
		Return(nil, repository.ErrConflict)

	resp, err := handler.CreateURLRecords(ctx, req)
	assert.Nil(t, resp)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
	assert.Contains(t, err.Error(), "URL conflict")
}

func TestCreateURLRecords_InternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockURLService := mocks.NewMockURLServiceIface(ctrl)

	handler := &grpc.ShortenerServer{
		Service: mockURLService,
		BaseURL: "http://localhost",
	}

	ctx := context.WithValue(context.Background(), middleware.UserIDKey, "user123")

	req := &pb.CreateURLRecordBatchRequest{
		Items: []*pb.CreateURLRecordBatchRequestItem{
			{CorrelationId: "c1", OriginalUrl: "https://example.com"},
		},
	}

	expectedInput := []models.BatchRequest{
		{CorrelationID: "c1", OriginalURL: "https://example.com"},
	}

	mockURLService.EXPECT().
		CreateURLRecords(gomock.Any(), expectedInput, "user123").
		Return(nil, errors.New("unexpected failure"))

	resp, err := handler.CreateURLRecords(ctx, req)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "unexpected failure")
}
