package verification

import (
	"context"
	"errors"
	"fmt"
	"time"

	deliveryv1 "github.com/Zhiruosama/Email-Service/gen/go/mailservice/delivery/v1"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store { return &Store{db: db} }

type deliveryEventRecord struct {
	EventID        string                    `gorm:"column:event_id;primaryKey"`
	MessageID      string                    `gorm:"column:message_id"`
	RequestID      string                    `gorm:"column:request_id"`
	Sequence       uint64                    `gorm:"column:sequence"`
	DeliveryStatus deliveryv1.DeliveryStatus `gorm:"column:delivery_status"`
	OccurredAt     time.Time                 `gorm:"column:occurred_at"`
	CreatedAt      time.Time                 `gorm:"column:created_at"`
}

func (deliveryEventRecord) TableName() string { return "email_delivery_events" }

func (s *Store) CreatePending(ctx context.Context, challenge *Challenge) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&Challenge{}).
			Where("email_fingerprint = ? AND purpose = ? AND state IN ?", challenge.EmailFingerprint, challenge.Purpose, []State{StatePendingDispatch, StateActive}).
			Updates(map[string]any{"state": StateTerminated, "updated_at": time.Now().UTC()}).Error; err != nil {
			return err
		}
		return tx.Create(challenge).Error
	})
}

func (s *Store) ApplySnapshot(ctx context.Context, snapshot DeliverySnapshot, validFor time.Duration) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var challenge Challenge
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("request_id = ?", snapshot.RequestID).First(&challenge).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrChallengeNotFound
			}
			return err
		}
		if challenge.MessageID != nil && *challenge.MessageID != snapshot.MessageID {
			return ErrChallengeConflict
		}
		updates := map[string]any{"message_id": snapshot.MessageID}
		if snapshot.LatestSequence >= challenge.LatestSequence {
			updates["latest_sequence"] = snapshot.LatestSequence
			updates["latest_delivery_status"] = snapshot.Status
			applyStatus(&challenge, snapshot.Status, snapshot.OccurredAt, validFor, updates)
		}
		return tx.Model(&challenge).Updates(updates).Error
	})
}

func (s *Store) ApplyEvent(ctx context.Context, event DeliveryEvent, validFor time.Duration) (EventDisposition, error) {
	if event.EventID == "" || event.MessageID == "" || event.RequestID == "" || event.Sequence == 0 || event.OccurredAt.IsZero() {
		return 0, ErrInvalidDeliveryEvent
	}

	disposition := EventAccepted
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var challenge Challenge
		query := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("request_id = ? OR message_id = ?", event.RequestID, event.MessageID)
		if err := query.First(&challenge).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrChallengeNotFound
			}
			return err
		}
		if challenge.RequestID != event.RequestID || (challenge.MessageID != nil && *challenge.MessageID != event.MessageID) {
			return ErrChallengeConflict
		}

		var existingEvent deliveryEventRecord
		eventErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("event_id = ?", event.EventID).First(&existingEvent).Error
		if eventErr == nil {
			disposition = EventDuplicate
			return nil
		}
		if !errors.Is(eventErr, gorm.ErrRecordNotFound) {
			return eventErr
		}

		record := &deliveryEventRecord{
			EventID: event.EventID, MessageID: event.MessageID, RequestID: event.RequestID,
			Sequence: event.Sequence, DeliveryStatus: event.Status, OccurredAt: event.OccurredAt.UTC(),
		}
		if err := tx.Create(record).Error; err != nil {
			return err
		}
		if event.Sequence <= challenge.LatestSequence {
			disposition = EventIgnoredStale
			return nil
		}

		updates := map[string]any{
			"message_id": event.MessageID, "latest_sequence": event.Sequence,
			"latest_delivery_status": event.Status,
		}
		applyStatus(&challenge, event.Status, event.OccurredAt, validFor, updates)
		return tx.Model(&challenge).Updates(updates).Error
	})
	return disposition, err
}

func (s *Store) VerifyAndConsume(ctx context.Context, fingerprint []byte, purpose Purpose, code string, digest func(string, string) []byte) error {
	var businessErr error
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var challenge Challenge
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("email_fingerprint = ? AND purpose = ? AND state = ?", fingerprint, purpose, StateActive).
			Order("active_at DESC").First(&challenge).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			businessErr = ErrCodeUnavailable
			return nil
		}
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if challenge.ExpiresAt == nil || !now.Before(*challenge.ExpiresAt) {
			if err := tx.Model(&challenge).Updates(map[string]any{"state": StateExpired, "updated_at": now}).Error; err != nil {
				return err
			}
			businessErr = ErrCodeUnavailable
			return nil
		}
		if !equalDigest(challenge.CodeDigest, digest(challenge.RequestID, code)) {
			attempts := challenge.FailedAttempts + 1
			updates := map[string]any{"failed_attempts": attempts, "updated_at": now}
			if attempts >= challenge.MaxAttempts {
				updates["state"] = StateLocked
				if err := tx.Model(&challenge).Updates(updates).Error; err != nil {
					return err
				}
				businessErr = ErrTooManyAttempts
				return nil
			}
			if err := tx.Model(&challenge).Updates(updates).Error; err != nil {
				return err
			}
			businessErr = ErrCodeMismatch
			return nil
		}
		return tx.Model(&challenge).Updates(map[string]any{"state": StateConsumed, "consumed_at": now, "updated_at": now}).Error
	})
	if err != nil {
		return err
	}
	return businessErr
}

func (s *Store) GetByRequestID(ctx context.Context, requestID string) (*Challenge, error) {
	var challenge Challenge
	err := s.db.WithContext(ctx).Where("request_id = ?", requestID).First(&challenge).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrChallengeNotFound
	}
	return &challenge, err
}

func (s *Store) MarkDeliveryFailed(ctx context.Context, requestID string) error {
	return s.db.WithContext(ctx).Model(&Challenge{}).
		Where("request_id = ? AND state = ?", requestID, StatePendingDispatch).
		Updates(map[string]any{"state": StateDeliveryFailed, "updated_at": time.Now().UTC()}).Error
}

func (s *Store) PendingForReconciliation(ctx context.Context, olderThan time.Time, limit int) ([]Challenge, error) {
	var challenges []Challenge
	err := s.db.WithContext(ctx).
		Where("state = ? AND created_at <= ?", StatePendingDispatch, olderThan.UTC()).
		Order("created_at ASC").Limit(limit).Find(&challenges).Error
	return challenges, err
}

func applyStatus(challenge *Challenge, status deliveryv1.DeliveryStatus, occurredAt time.Time, validFor time.Duration, updates map[string]any) {
	if challenge.State == StateConsumed || challenge.State == StateLocked || challenge.State == StateExpired || challenge.State == StateTerminated {
		return
	}
	switch status {
	case deliveryv1.DeliveryStatus_DELIVERY_STATUS_PROVIDER_ACCEPTED,
		deliveryv1.DeliveryStatus_DELIVERY_STATUS_DELIVERED:
		if challenge.State == StatePendingDispatch {
			activeAt := occurredAt.UTC()
			expiresAt := activeAt.Add(validFor)
			updates["state"] = StateActive
			updates["active_at"] = activeAt
			updates["expires_at"] = expiresAt
		}
	case deliveryv1.DeliveryStatus_DELIVERY_STATUS_BOUNCED,
		deliveryv1.DeliveryStatus_DELIVERY_STATUS_COMPLAINED,
		deliveryv1.DeliveryStatus_DELIVERY_STATUS_PERMANENTLY_FAILED,
		deliveryv1.DeliveryStatus_DELIVERY_STATUS_DEAD_LETTERED:
		updates["state"] = StateDeliveryFailed
	case deliveryv1.DeliveryStatus_DELIVERY_STATUS_CANCELED,
		deliveryv1.DeliveryStatus_DELIVERY_STATUS_EXPIRED,
		deliveryv1.DeliveryStatus_DELIVERY_STATUS_UNKNOWN_FINAL:
		updates["state"] = StateTerminated
	}
}

func (s DeliverySnapshot) validate() error {
	if s.MessageID == "" || s.RequestID == "" {
		return fmt.Errorf("%w: incomplete mail status", ErrDeliveryNotAccepted)
	}
	return nil
}
