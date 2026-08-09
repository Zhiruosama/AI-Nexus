package mailgateway

import (
	"context"
	"net"
	"testing"
	"time"

	deliveryv1 "github.com/Zhiruosama/Email-Service/gen/go/mailservice/delivery/v1"
	"github.com/Zhiruosama/ai_nexus/internal/verification"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type captureDeliveryServer struct {
	deliveryv1.UnimplementedDeliveryServiceServer
	request *deliveryv1.SubmitEmailRequest
}

func (s *captureDeliveryServer) SubmitEmail(_ context.Context, request *deliveryv1.SubmitEmailRequest) (*deliveryv1.SubmitEmailResponse, error) {
	s.request = request
	return &deliveryv1.SubmitEmailResponse{
		Disposition: deliveryv1.SubmitDisposition_SUBMIT_DISPOSITION_ACCEPTED,
		Message: &deliveryv1.EmailMessage{
			MessageId: "message-1", IdempotencyKey: request.GetIdempotencyKey(),
			Status:    deliveryv1.DeliveryStatus_DELIVERY_STATUS_ACCEPTED,
			UpdatedAt: timestamppb.Now(), LatestSequence: 1,
		},
	}, nil
}

func TestSubmitVerificationUsesCanonicalV01Contract(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	capture := &captureDeliveryServer{}
	deliveryv1.RegisterDeliveryServiceServer(server, capture)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)

	client, err := NewClient(Config{
		Address: listener.Addr().String(), Timeout: time.Second, AllowInsecure: true,
		SenderIdentityKey: "ainexus.default", Locale: "zh-CN", DispatchDeadline: 2 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	createdAt := time.Now().UTC().Truncate(time.Second)
	requestID := "c4133024-289f-402d-8c19-8616ba396c7d"
	_, err = client.SubmitVerification(context.Background(), verification.VerificationEmail{
		RequestID: requestID, Recipient: "person@example.com", Code: "123456",
		Purpose: verification.PurposeLogin, ValidForSeconds: 300, CreatedAt: createdAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	request := capture.request
	if request.GetIdempotencyKey() != requestID || request.GetRecipient().GetEmail() != "person@example.com" {
		t.Fatalf("unexpected request identity")
	}
	if request.GetSenderIdentityKey() != "ainexus.default" || request.GetContent().GetTemplate().GetKey() != "verification_code.v1" || request.GetContent().GetLocale() != "zh-CN" {
		t.Fatalf("unexpected content contract: %#v", request.GetContent())
	}
	variables := request.GetContent().GetVariables().AsMap()
	if variables["code"] != "123456" || variables["purpose"] != "LOGIN" || variables["valid_for_seconds"] != float64(300) || len(variables) != 3 {
		t.Fatalf("unexpected variables: %#v", variables)
	}
	if request.GetCategory() != deliveryv1.EmailCategory_EMAIL_CATEGORY_CRITICAL || request.GetPriority() != 9 || request.GetDuplicateRiskPolicy() != deliveryv1.DuplicateRiskPolicy_DUPLICATE_RISK_POLICY_AVOID_DUPLICATE {
		t.Fatalf("unexpected delivery policy")
	}
	if got := request.GetDispatchDeadline().AsTime(); !got.Equal(createdAt.Add(2 * time.Minute)) {
		t.Fatalf("dispatch deadline = %s", got)
	}
	if request.GetMetadata() != nil {
		t.Fatalf("metadata must remain empty until Mail Service canonicalizes non-empty JSON consistently")
	}
}
