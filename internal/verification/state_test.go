package verification

import (
	"testing"
	"time"

	deliveryv1 "github.com/Zhiruosama/Email-Service/gen/go/mailservice/delivery/v1"
)

func TestApplyStatusActivatesOnceWithoutExtendingExpiry(t *testing.T) {
	challenge := &Challenge{State: StatePendingDispatch}
	first := time.Date(2026, 8, 9, 10, 0, 0, 0, time.UTC)
	updates := map[string]any{}
	applyStatus(challenge, deliveryv1.DeliveryStatus_DELIVERY_STATUS_PROVIDER_ACCEPTED, first, 5*time.Minute, updates)
	if updates["state"] != StateActive || updates["active_at"] != first || updates["expires_at"] != first.Add(5*time.Minute) {
		t.Fatalf("unexpected activation updates: %#v", updates)
	}

	challenge.State = StateActive
	laterUpdates := map[string]any{}
	applyStatus(challenge, deliveryv1.DeliveryStatus_DELIVERY_STATUS_DELIVERED, first.Add(time.Minute), 5*time.Minute, laterUpdates)
	if _, exists := laterUpdates["expires_at"]; exists {
		t.Fatalf("delivery event extended an active challenge: %#v", laterUpdates)
	}
}

func TestApplyStatusTerminalMappings(t *testing.T) {
	tests := []struct {
		status deliveryv1.DeliveryStatus
		want   State
	}{
		{deliveryv1.DeliveryStatus_DELIVERY_STATUS_BOUNCED, StateDeliveryFailed},
		{deliveryv1.DeliveryStatus_DELIVERY_STATUS_PERMANENTLY_FAILED, StateDeliveryFailed},
		{deliveryv1.DeliveryStatus_DELIVERY_STATUS_CANCELED, StateTerminated},
		{deliveryv1.DeliveryStatus_DELIVERY_STATUS_UNKNOWN_FINAL, StateTerminated},
	}
	for _, test := range tests {
		t.Run(test.status.String(), func(t *testing.T) {
			updates := map[string]any{}
			applyStatus(&Challenge{State: StatePendingDispatch}, test.status, time.Now(), time.Minute, updates)
			if got := updates["state"]; got != test.want {
				t.Fatalf("state = %v, want %v", got, test.want)
			}
		})
	}
}

func TestGenerateCodeAlwaysHasSixDigits(t *testing.T) {
	for range 100 {
		code, err := generateCode()
		if err != nil {
			t.Fatal(err)
		}
		if len(code) != 6 {
			t.Fatalf("code length = %d", len(code))
		}
		for _, character := range code {
			if character < '0' || character > '9' {
				t.Fatalf("code contains a non-digit")
			}
		}
	}
}
