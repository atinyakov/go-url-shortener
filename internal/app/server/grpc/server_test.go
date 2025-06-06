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
	"github.com/atinyakov/go-url-shortener/internal/app/service"
	"github.com/atinyakov/go-url-shortener/internal/intercepters"
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

func TestDeleteBatch_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockURLServiceIface(ctrl)
	handler := &grpc.ShortenerServer{Service: mockService}

	// Capture input for assertion.
	var capturedRecords []storage.URLRecord
	deleteCalled := make(chan struct{}, 1)

	// Override the global variable to avoid running real goroutine.
	grpc.СallDeleteURLRecords = func(service service.URLServiceIface, ctx context.Context, records []storage.URLRecord) {
		capturedRecords = records
		deleteCalled <- struct{}{}
	}

	ctx := context.WithValue(context.Background(), middleware.UserIDKey, "user123")

	req := &pb.DeleteURLRecordsRequest{
		Items: []*pb.Short{
			{Short: "abc123"},
			{Short: "def456"},
		},
	}

	err := handler.DeleteBatch(ctx, req)
	assert.NoError(t, err)

	// Wait for fake goroutine substitute to be called
	<-deleteCalled

	assert.Len(t, capturedRecords, 2)
	assert.Equal(t, "abc123", capturedRecords[0].Short)
	assert.Equal(t, "def456", capturedRecords[1].Short)
	assert.Equal(t, "user123", capturedRecords[0].UserID)
}

func TestDeleteBatch_MissingUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockURLServiceIface(ctrl)
	handler := &grpc.ShortenerServer{Service: mockService}

	ctx := context.Background() // No user ID set
	req := &pb.DeleteURLRecordsRequest{
		Items: []*pb.Short{{Short: "abc123"}},
	}

	err := handler.DeleteBatch(ctx, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user ID missing")
}

func TestDeleteBatch_EmptyList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockURLServiceIface(ctrl)
	handler := &grpc.ShortenerServer{Service: mockService}

	deleteCalled := make(chan struct{}, 1)

	grpc.СallDeleteURLRecords = func(service service.URLServiceIface, ctx context.Context, records []storage.URLRecord) {
		assert.Empty(t, records)
		deleteCalled <- struct{}{}
	}

	ctx := context.WithValue(context.Background(), middleware.UserIDKey, "user123")

	req := &pb.DeleteURLRecordsRequest{
		Items: []*pb.Short{},
	}

	err := handler.DeleteBatch(ctx, req)
	assert.NoError(t, err)
	<-deleteCalled
}

func TestGetURLByShort(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockURLServiceIface(ctrl)
	handler := &grpc.ShortenerServer{Service: mockSvc}

	ctx := context.Background()
	req := &pb.Short{Short: "abc123"}
	expected := &storage.URLRecord{Original: "https://example.com"}

	mockSvc.EXPECT().GetURLByShort(ctx, req.Short).Return(expected, nil)

	resp, err := handler.GetURLByShort(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, expected.Original, resp.Original)
}

func TestGetURLByShort_Deleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockURLServiceIface(ctrl)
	handler := &grpc.ShortenerServer{Service: mockSvc}

	ctx := context.Background()
	req := &pb.Short{Short: "abc123"}
	expected := &storage.URLRecord{Original: "123", IsDeleted: true}

	mockSvc.EXPECT().GetURLByShort(ctx, req.Short).Return(expected, nil)

	resp, err := handler.GetURLByShort(ctx, req)
	assert.NoError(t, err)
	assert.True(t, resp.IsDeleted)
}

func TestGetURLByUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockURLServiceIface(ctrl)
	handler := &grpc.ShortenerServer{Service: mockSvc}

	ctx := context.WithValue(context.Background(), middleware.UserIDKey, "user123")
	expected := &[]models.ByIDRequest{
		{OriginalURL: "https://site1.com", ShortURL: "abc1"},
		{OriginalURL: "https://site2.com", ShortURL: "abc2"},
	}

	mockSvc.EXPECT().GetURLByUserID(ctx, "user123").Return(expected, nil)

	resp, err := handler.GetURLByUserID(ctx, &pb.ID{})
	assert.NoError(t, err)
	assert.Len(t, resp.Items, 2)
	assert.Equal(t, "https://site1.com", resp.Items[0].OriginalUrl)
}

func TestPingContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockURLServiceIface(ctrl)
	handler := &grpc.ShortenerServer{Service: mockSvc}

	ctx := context.Background()
	mockSvc.EXPECT().PingContext(ctx).Return(nil)

	_, err := handler.PingContext(ctx, &emptypb.Empty{})
	assert.NoError(t, err)
}

func TestGetStats_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockURLServiceIface(ctrl)
	handler := &grpc.ShortenerServer{
		Service:       mockSvc,
		TrustedSubnet: "10.0.0.1",
	}

	ctx := context.WithValue(context.Background(), intercepters.RealIPKey, "10.0.0.1")
	mockSvc.EXPECT().GetStats(ctx).Return(&models.StatsResponse{Urls: 42, Users: 5}, nil)

	resp, err := handler.GetStats(ctx, &emptypb.Empty{})
	assert.NoError(t, err)
	assert.Equal(t, int32(42), resp.Urls)
	assert.Equal(t, int32(5), resp.Users)
}

func TestGetStats_Unauthorized_MissingHeader(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := &grpc.ShortenerServer{TrustedSubnet: "10.0.0.1"}

	ctx := context.Background() // no RealIPKey
	_, err := handler.GetStats(ctx, &emptypb.Empty{})
	assert.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	assert.Contains(t, err.Error(), "X-Real-IP header missing")
}

func TestGetStats_Unauthorized_WrongSubnet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := &grpc.ShortenerServer{TrustedSubnet: "10.0.0.1"}
	ctx := context.WithValue(context.Background(), intercepters.RealIPKey, "192.168.1.1")

	_, err := handler.GetStats(ctx, &emptypb.Empty{})
	assert.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	assert.Contains(t, err.Error(), "subnet is not trusted")
}
