package mailcallback

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"

	deliveryv1 "github.com/Zhiruosama/Email-Service/gen/go/mailservice/delivery/v1"
	"github.com/Zhiruosama/ai_nexus/internal/verification"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

type Config struct {
	Address       string
	AllowInsecure bool
	TLSCertFile   string
	TLSKeyFile    string
}

type EventProcessor interface {
	ApplyDeliveryEvent(context.Context, verification.DeliveryEvent) (verification.EventDisposition, error)
}

type Server struct {
	deliveryv1.UnimplementedDeliveryEventReceiverServiceServer
	grpcServer *grpc.Server
	listener   net.Listener
	service    EventProcessor
}

func NewServer(cfg Config, service EventProcessor) (*Server, error) {
	if cfg.Address == "" || service == nil {
		return nil, fmt.Errorf("invalid mail callback configuration")
	}
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return nil, fmt.Errorf("listen for mail callbacks: %w", err)
	}
	options := make([]grpc.ServerOption, 0, 1)
	if !cfg.AllowInsecure {
		certificate, certErr := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
		if certErr != nil {
			_ = listener.Close()
			return nil, fmt.Errorf("load mail callback TLS identity: %w", certErr)
		}
		options = append(options, grpc.Creds(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{certificate}})))
	}
	server := &Server{grpcServer: grpc.NewServer(options...), listener: listener, service: service}
	deliveryv1.RegisterDeliveryEventReceiverServiceServer(server.grpcServer, server)
	return server, nil
}

func (s *Server) Serve() error {
	err := s.grpcServer.Serve(s.listener)
	if errors.Is(err, grpc.ErrServerStopped) {
		return nil
	}
	return err
}

func (s *Server) Stop() { s.grpcServer.GracefulStop() }

func (s *Server) ReportDeliveryEvent(ctx context.Context, request *deliveryv1.ReportDeliveryEventRequest) (*deliveryv1.ReportDeliveryEventResponse, error) {
	event := request.GetEvent()
	if event == nil || event.GetStatus() == deliveryv1.DeliveryStatus_DELIVERY_STATUS_UNSPECIFIED || event.GetOccurredAt() == nil {
		return nil, status.Error(codes.InvalidArgument, "delivery event is incomplete")
	}
	if err := event.GetOccurredAt().CheckValid(); err != nil {
		return nil, status.Error(codes.InvalidArgument, "delivery event timestamp is invalid")
	}
	disposition, err := s.service.ApplyDeliveryEvent(ctx, verification.DeliveryEvent{
		EventID: event.GetEventId(), MessageID: event.GetMessageId(), RequestID: event.GetIdempotencyKey(),
		Status: event.GetStatus(), Sequence: event.GetSequence(), AttemptNumber: event.GetAttemptNumber(),
		OccurredAt: event.GetOccurredAt().AsTime(),
	})
	if err != nil {
		switch {
		case errors.Is(err, verification.ErrInvalidDeliveryEvent):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errors.Is(err, verification.ErrChallengeNotFound):
			return nil, status.Error(codes.NotFound, err.Error())
		case errors.Is(err, verification.ErrChallengeConflict):
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		default:
			return nil, status.Error(codes.Internal, "delivery event could not be persisted")
		}
	}
	ack := deliveryv1.EventAckDisposition_EVENT_ACK_DISPOSITION_ACCEPTED
	if disposition == verification.EventDuplicate {
		ack = deliveryv1.EventAckDisposition_EVENT_ACK_DISPOSITION_DUPLICATE
	} else if disposition == verification.EventIgnoredStale {
		ack = deliveryv1.EventAckDisposition_EVENT_ACK_DISPOSITION_IGNORED_STALE
	}
	return &deliveryv1.ReportDeliveryEventResponse{EventId: event.GetEventId(), Disposition: ack}, nil
}
