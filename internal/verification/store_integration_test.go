//go:build integration

package verification

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"testing"
	"time"

	deliveryv1 "github.com/Zhiruosama/Email-Service/gen/go/mailservice/delivery/v1"
	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestStoreDeliveryEventsAndOneTimeConsumption(t *testing.T) {
	dsn := os.Getenv("AI_NEXUS_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("AI_NEXUS_TEST_MYSQL_DSN is not set")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(db)
	ctx := context.Background()
	requestID := uuid.NewString()
	messageID := "test-message-" + uuid.NewString()
	eventID := "test-event-" + uuid.NewString()
	fingerprint := sha256.Sum256([]byte(requestID))
	digest := func(id, code string) []byte {
		value := sha256.Sum256([]byte(id + ":" + code))
		return value[:]
	}
	t.Cleanup(func() {
		db.Exec("DELETE FROM email_delivery_events WHERE request_id = ?", requestID)
		db.Exec("DELETE FROM email_verification_challenges WHERE request_id = ?", requestID)
	})

	now := time.Now().UTC().Truncate(time.Microsecond)
	challenge := &Challenge{
		RequestID: requestID, EmailFingerprint: fingerprint[:], Purpose: PurposeLogin,
		CodeDigest: digest(requestID, "123456"), State: StatePendingDispatch,
		ValidForSeconds: 300, MaxAttempts: 5, CreatedAt: now, UpdatedAt: now,
	}
	if err = store.CreatePending(ctx, challenge); err != nil {
		t.Fatal(err)
	}
	if err = store.ApplySnapshot(ctx, DeliverySnapshot{
		MessageID: messageID, RequestID: requestID,
		Status: deliveryv1.DeliveryStatus_DELIVERY_STATUS_ACCEPTED, LatestSequence: 1, OccurredAt: now,
	}, 5*time.Minute); err != nil {
		t.Fatal(err)
	}

	event := DeliveryEvent{
		EventID: eventID, MessageID: messageID, RequestID: requestID,
		Status:   deliveryv1.DeliveryStatus_DELIVERY_STATUS_PROVIDER_ACCEPTED,
		Sequence: 4, OccurredAt: now.Add(time.Second),
	}
	if disposition, applyErr := store.ApplyEvent(ctx, event, 5*time.Minute); applyErr != nil || disposition != EventAccepted {
		t.Fatalf("first event = (%v, %v)", disposition, applyErr)
	}
	if disposition, applyErr := store.ApplyEvent(ctx, event, 5*time.Minute); applyErr != nil || disposition != EventDuplicate {
		t.Fatalf("duplicate event = (%v, %v)", disposition, applyErr)
	}
	stale := event
	stale.EventID = "test-event-" + uuid.NewString()
	stale.Sequence = 3
	if disposition, applyErr := store.ApplyEvent(ctx, stale, 5*time.Minute); applyErr != nil || disposition != EventIgnoredStale {
		t.Fatalf("stale event = (%v, %v)", disposition, applyErr)
	}

	if err = store.VerifyAndConsume(ctx, fingerprint[:], PurposeLogin, "000000", digest); !errors.Is(err, ErrCodeMismatch) {
		t.Fatalf("wrong code error = %v", err)
	}
	if err = store.VerifyAndConsume(ctx, fingerprint[:], PurposeLogin, "123456", digest); err != nil {
		t.Fatalf("consume correct code: %v", err)
	}
	if err = store.VerifyAndConsume(ctx, fingerprint[:], PurposeLogin, "123456", digest); !errors.Is(err, ErrCodeUnavailable) {
		t.Fatalf("second consumption error = %v", err)
	}

	saved, err := store.GetByRequestID(ctx, requestID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.State != StateConsumed || saved.FailedAttempts != 1 || saved.LatestSequence != 4 {
		t.Fatalf("unexpected saved challenge: state=%s attempts=%d sequence=%d", saved.State, saved.FailedAttempts, saved.LatestSequence)
	}
}

func TestStoreLocksChallengeAfterMaximumFailures(t *testing.T) {
	dsn := os.Getenv("AI_NEXUS_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("AI_NEXUS_TEST_MYSQL_DSN is not set")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(db)
	ctx := context.Background()
	requestID := uuid.NewString()
	fingerprint := sha256.Sum256([]byte(requestID))
	digest := func(id, code string) []byte {
		value := sha256.Sum256([]byte(id + ":" + code))
		return value[:]
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	expiresAt := now.Add(5 * time.Minute)
	challenge := &Challenge{
		RequestID: requestID, EmailFingerprint: fingerprint[:], Purpose: PurposeResetPassword,
		CodeDigest: digest(requestID, "123456"), State: StateActive, ValidForSeconds: 300,
		MaxAttempts: 2, ActiveAt: &now, ExpiresAt: &expiresAt, CreatedAt: now, UpdatedAt: now,
	}
	if err = store.CreatePending(ctx, challenge); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec("DELETE FROM email_verification_challenges WHERE request_id = ?", requestID) })

	if err = store.VerifyAndConsume(ctx, fingerprint[:], PurposeResetPassword, "000000", digest); !errors.Is(err, ErrCodeMismatch) {
		t.Fatalf("first failure = %v", err)
	}
	if err = store.VerifyAndConsume(ctx, fingerprint[:], PurposeResetPassword, "000001", digest); !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("second failure = %v", err)
	}
	saved, err := store.GetByRequestID(ctx, requestID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.State != StateLocked || saved.FailedAttempts != 2 {
		t.Fatalf("state=%s attempts=%d", saved.State, saved.FailedAttempts)
	}
}
