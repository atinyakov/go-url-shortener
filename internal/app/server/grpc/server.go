package grpc

import (
	"context"
	"errors"
	"fmt"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/atinyakov/go-url-shortener/proto"

	"github.com/atinyakov/go-url-shortener/internal/app/service"
	"github.com/atinyakov/go-url-shortener/internal/intercepters"
	"github.com/atinyakov/go-url-shortener/internal/middleware"
	"github.com/atinyakov/go-url-shortener/internal/models"
	"github.com/atinyakov/go-url-shortener/internal/repository"
	"github.com/atinyakov/go-url-shortener/internal/storage"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
)

// Server wraps the gRPC server and dependencies.
type Server struct {
	grpcServer *grpc.Server
	port       int
	service    service.URLServiceIface
	logger     *zap.Logger
}

// New creates a new gRPC server instance.
func New(baseURL string, trustedSubnet string, logger *zap.Logger, svc *service.URLService, port int) *Server {
	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			logging.UnaryServerInterceptor(intercepters.InterceptorLogger(logger)),
			intercepters.WithJWT(service.NewAuth(svc)),
		),
	)

	// Register the gRPC service implementation
	pb.RegisterURLServiceServer(s, &shortenerServer{
		service: svc,
		baseURL: baseURL,
	})

	return &Server{
		grpcServer: s,
		port:       port,
		service:    svc,
		logger:     logger,
	}
}

// Start runs the gRPC server.
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		s.logger.Error("gRPC server failed to listen:", zap.Error(err))
		return err
	}

	s.logger.Info("gRPC server listening on port", zap.Int("port", s.port))
	return s.grpcServer.Serve(lis)
}

// GracefulStop shuts down the server gracefully.
func (s *Server) GracefulStop() {
	s.grpcServer.GracefulStop()
}

// --- Implementation of the gRPC interface ---

type shortenerServer struct {
	pb.UnimplementedURLServiceServer
	service *service.URLService
	baseURL string
}

// CreateURLRecord
func (s *shortenerServer) CreateURLRecord(ctx context.Context, req *pb.CreateURLRecordRequest) (*pb.CreateURLRecordResponse, error) {
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		return nil, status.Error(codes.Internal, "user ID missing in context")
	}

	record, err := s.service.CreateURLRecord(ctx, req.Url, userID)
	if err != nil {
		return nil, err
	}

	return &pb.CreateURLRecordResponse{
		Result: s.baseURL + "/" + record.Short,
	}, nil
}

func (s *shortenerServer) CreateURLRecords(ctx context.Context, req *pb.CreateURLRecordBatchRequest) (*pb.CreateURLRecordBatchResponse, error) {
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		return nil, status.Error(codes.Internal, "user ID missing in context")
	}

	var urlsR []models.BatchRequest

	for _, item := range req.Items {
		urlsR = append(urlsR, models.BatchRequest{
			CorrelationID: item.CorrelationId,
			OriginalURL:   item.OriginalUrl,
		})
	}

	batchUrls, err := s.service.CreateURLRecords(ctx, urlsR, userID)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, status.Error(codes.AlreadyExists, "URL conflict")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	var respItems []*pb.CreateURLRecordBatchResponceItem
	for _, b := range *batchUrls {
		respItems = append(respItems, &pb.CreateURLRecordBatchResponceItem{
			CorrelationId: b.CorrelationID,
			ShortUrl:      b.ShortURL,
		})
	}

	return &pb.CreateURLRecordBatchResponse{
		Items: respItems,
	}, nil
}

// callDeleteURLRecords is separated to allow override in tests.
var СallDeleteURLRecords = func(service service.URLServiceIface, ctx context.Context, records []storage.URLRecord) {
	go service.DeleteURLRecords(ctx, records)
}

// DeleteBatch handles DELETE requests for deleting multiple URLs in batch.
// It reads a list of shortened URLs from the request body and deletes them asynchronously.
func (s *shortenerServer) DeleteBatch(ctx context.Context, req *pb.DeleteURLRecordsRequest) error {
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		return status.Error(codes.Internal, "user ID missing in context")
	}

	// Prepare the list of URLs to delete.
	var toDelete []storage.URLRecord
	for _, url := range req.Items {
		toDelete = append(toDelete, storage.URLRecord{Short: url.Short, UserID: userID})
	}

	// Perform the deletion asynchronously.
	СallDeleteURLRecords(s.service, ctx, toDelete)

	return nil
}

func (s *shortenerServer) GetURLByShort(ctx context.Context, req *pb.Short) (*pb.URLRecord, error) {
	// Resolve the original URL using the service.
	r, err := s.service.GetURLByShort(ctx, req.Short)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	// Check if the URL is marked as deleted.
	if r.IsDeleted {
		return &pb.URLRecord{
			IsDeleted: r.IsDeleted,
		}, nil
	}

	return &pb.URLRecord{
		Original: r.Original,
	}, nil
}

func (s *shortenerServer) GetURLByUserID(ctx context.Context, req *pb.ID) (*pb.ByUserIDResponse, error) {
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		return nil, status.Error(codes.Internal, "user ID missing in context")
	}

	// Retrieve the URLs associated with the user from the service.
	urls, err := s.service.GetURLByUserID(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var respItems []*pb.ByUserID
	for _, b := range *urls {
		respItems = append(respItems, &pb.ByUserID{
			OriginalUrl: b.OriginalURL,
			ShortUrl:    b.ShortURL,
		})
	}

	return &pb.ByUserIDResponse{
		Items: respItems,
	}, nil
}
