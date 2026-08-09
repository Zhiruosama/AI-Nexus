package mailcallback

import (
	"context"
	"testing"
	"time"

	deliveryv1 "github.com/Zhiruosama/Email-Service/gen/go/mailservice/delivery/v1"
	"github.com/Zhiruosama/ai_nexus/internal/verification"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type stubProcessor struct {
	disposition verification.EventDisposition
	err         error
	event       verification.DeliveryEvent
}

func (p *stubProcessor) ApplyDeliveryEvent(_ context.Context, event verification.DeliveryEvent) (verification.EventDisposition, error) {
	p.event = event
	return p.disposition, p.err
}

func TestReportDeliveryEventMapsAcknowledgement(t *testing.T) {
	tests := []struct {
		name string
		got  verification.EventDisposition
		want deliveryv1.EventAckDisposition
	}{
		{"accepted", verification.EventAccepted, deliveryv1.EventAckDisposition_EVENT_ACK_DISPOSITION_ACCEPTED},
		{"duplicate", verification.EventDuplicate, deliveryv1.EventAckDisposition_EVENT_ACK_DISPOSITION_DUPLICATE},
		{"stale", verification.EventIgnoredStale, deliveryv1.EventAckDisposition_EVENT_ACK_DISPOSITION_IGNORED_STALE},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			processor := &stubProcessor{disposition: test.got}
			server := &Server{service: processor}
			response, err := server.ReportDeliveryEvent(context.Background(), validRequest())
			if err != nil {
				t.Fatal(err)
			}
			if response.GetDisposition() != test.want || processor.event.RequestID != "request-1" {
				t.Fatalf("response = %#v, event = %#v", response, processor.event)
			}
		})
	}
}

func TestReportDeliveryEventRejectsIncompleteEvent(t *testing.T) {
	server := &Server{service: &stubProcessor{}}
	_, err := server.ReportDeliveryEvent(context.Background(), &deliveryv1.ReportDeliveryEventRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %s, want InvalidArgument", status.Code(err))
	}
}

func validRequest() *deliveryv1.ReportDeliveryEventRequest {
	return &deliveryv1.ReportDeliveryEventRequest{Event: &deliveryv1.DeliveryEvent{
		EventId: "event-1", MessageId: "message-1", IdempotencyKey: "request-1",
		Status:     deliveryv1.DeliveryStatus_DELIVERY_STATUS_PROVIDER_ACCEPTED,
		OccurredAt: timestamppb.New(time.Now().UTC()), Sequence: 4,
	}}
}
