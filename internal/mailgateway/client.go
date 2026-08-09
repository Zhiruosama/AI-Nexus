package mailgateway

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	deliveryv1 "github.com/Zhiruosama/Email-Service/gen/go/mailservice/delivery/v1"
	"github.com/Zhiruosama/ai_nexus/internal/verification"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Config struct {
	Address           string
	Timeout           time.Duration
	AllowInsecure     bool
	TLSCAFile         string
	TLSServerName     string
	SenderIdentityKey string
	Locale            string
	DispatchDeadline  time.Duration
}

type Client struct {
	conn   *grpc.ClientConn
	client deliveryv1.DeliveryServiceClient
	config Config
}

func NewClient(cfg Config) (*Client, error) {
	if cfg.Address == "" || cfg.Timeout <= 0 || cfg.SenderIdentityKey == "" || cfg.Locale == "" || cfg.DispatchDeadline <= 0 {
		return nil, fmt.Errorf("invalid mail service client configuration")
	}
	transport, err := transportCredentials(cfg)
	if err != nil {
		return nil, err
	}
	conn, err := grpc.NewClient(cfg.Address, grpc.WithTransportCredentials(transport))
	if err != nil {
		return nil, fmt.Errorf("create mail service connection: %w", err)
	}
	return &Client{conn: conn, client: deliveryv1.NewDeliveryServiceClient(conn), config: cfg}, nil
}

func (c *Client) Close() error { return c.conn.Close() }

func (c *Client) SubmitVerification(ctx context.Context, email verification.VerificationEmail) (verification.DeliverySnapshot, error) {
	variables, err := structpb.NewStruct(map[string]any{
		"code": email.Code, "purpose": string(email.Purpose), "valid_for_seconds": email.ValidForSeconds,
	})
	if err != nil {
		return verification.DeliverySnapshot{}, fmt.Errorf("build verification template variables: %w", err)
	}
	request := &deliveryv1.SubmitEmailRequest{
		IdempotencyKey:    email.RequestID,
		Recipient:         &deliveryv1.Recipient{Email: email.Recipient},
		SenderIdentityKey: c.config.SenderIdentityKey,
		Content: &deliveryv1.EmailContent{
			Template: &deliveryv1.TemplateReference{Key: "verification_code.v1"},
			Locale:   c.config.Locale, Variables: variables,
		},
		Category:            deliveryv1.EmailCategory_EMAIL_CATEGORY_CRITICAL,
		Priority:            9,
		DispatchDeadline:    timestamppb.New(email.CreatedAt.Add(c.config.DispatchDeadline)),
		DuplicateRiskPolicy: deliveryv1.DuplicateRiskPolicy_DUPLICATE_RISK_POLICY_AVOID_DUPLICATE,
	}
	callCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()
	response, err := c.client.SubmitEmail(callCtx, request)
	if err != nil {
		switch status.Code(err) {
		case codes.DeadlineExceeded, codes.Unavailable, codes.Unknown:
			return verification.DeliverySnapshot{}, fmt.Errorf("%w: %s", verification.ErrSubmissionUnknown, status.Code(err))
		}
		return verification.DeliverySnapshot{}, err
	}
	if response.GetDisposition() != deliveryv1.SubmitDisposition_SUBMIT_DISPOSITION_ACCEPTED &&
		response.GetDisposition() != deliveryv1.SubmitDisposition_SUBMIT_DISPOSITION_DUPLICATE {
		return verification.DeliverySnapshot{}, fmt.Errorf("unexpected submit disposition: %s", response.GetDisposition())
	}
	return snapshotFromMessage(response.GetMessage())
}

func (c *Client) GetByRequestID(ctx context.Context, requestID string) (verification.DeliverySnapshot, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()
	response, err := c.client.GetEmail(callCtx, &deliveryv1.GetEmailRequest{
		Selector: &deliveryv1.GetEmailRequest_IdempotencyKey{IdempotencyKey: requestID},
	})
	if err != nil {
		return verification.DeliverySnapshot{}, err
	}
	return snapshotFromMessage(response.GetMessage())
}

func snapshotFromMessage(message *deliveryv1.EmailMessage) (verification.DeliverySnapshot, error) {
	if message == nil {
		return verification.DeliverySnapshot{}, fmt.Errorf("mail service returned no message")
	}
	occurredAt := time.Now().UTC()
	if message.GetUpdatedAt() != nil && message.GetUpdatedAt().IsValid() {
		occurredAt = message.GetUpdatedAt().AsTime()
	} else if message.GetAcceptedAt() != nil && message.GetAcceptedAt().IsValid() {
		occurredAt = message.GetAcceptedAt().AsTime()
	}
	return verification.DeliverySnapshot{
		MessageID: message.GetMessageId(), RequestID: message.GetIdempotencyKey(),
		Status: message.GetStatus(), LatestSequence: message.GetLatestSequence(), OccurredAt: occurredAt,
	}, nil
}

func transportCredentials(cfg Config) (credentials.TransportCredentials, error) {
	if cfg.AllowInsecure {
		return insecure.NewCredentials(), nil
	}
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12, ServerName: cfg.TLSServerName}
	if cfg.TLSCAFile != "" {
		pem, err := os.ReadFile(cfg.TLSCAFile)
		if err != nil {
			return nil, fmt.Errorf("read mail service CA: %w", err)
		}
		pool, err := x509.SystemCertPool()
		if err != nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("mail service CA file contains no certificates")
		}
		tlsConfig.RootCAs = pool
	}
	return credentials.NewTLS(tlsConfig), nil
}
