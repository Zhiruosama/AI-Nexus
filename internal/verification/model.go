package verification

import (
	"context"
	"errors"
	"time"

	deliveryv1 "github.com/Zhiruosama/Email-Service/gen/go/mailservice/delivery/v1"
)

type Purpose string

const (
	PurposeRegister      Purpose = "REGISTER"
	PurposeResetPassword Purpose = "RESET_PASSWORD"
	PurposeLogin         Purpose = "LOGIN"
)

func PurposeFromCode(code int) (Purpose, error) {
	switch code {
	case 1:
		return PurposeRegister, nil
	case 2:
		return PurposeResetPassword, nil
	case 3:
		return PurposeLogin, nil
	default:
		return "", ErrInvalidPurpose
	}
}

type State string

const (
	StatePendingDispatch State = "PENDING_DISPATCH"
	StateActive          State = "ACTIVE"
	StateConsumed        State = "CONSUMED"
	StateLocked          State = "LOCKED"
	StateExpired         State = "EXPIRED"
	StateDeliveryFailed  State = "DELIVERY_FAILED"
	StateTerminated      State = "TERMINATED"
)

var (
	ErrInvalidPurpose       = errors.New("invalid verification purpose")
	ErrInvalidRequestID     = errors.New("invalid idempotency key")
	ErrSubmissionUnknown    = errors.New("mail submission outcome is unknown")
	ErrCooldown             = errors.New("verification code request is too frequent")
	ErrDeliveryNotAccepted  = errors.New("mail delivery request was not accepted")
	ErrChallengeNotFound    = errors.New("verification challenge not found")
	ErrCodeUnavailable      = errors.New("verification code is not active or has expired")
	ErrCodeMismatch         = errors.New("verification code is incorrect")
	ErrTooManyAttempts      = errors.New("verification code has been locked")
	ErrChallengeConflict    = errors.New("verification challenge delivery identity conflict")
	ErrInvalidDeliveryEvent = errors.New("invalid delivery event")
)

type Challenge struct {
	ID                   uint64                    `gorm:"column:id;primaryKey"`
	RequestID            string                    `gorm:"column:request_id"`
	MessageID            *string                   `gorm:"column:message_id"`
	EmailFingerprint     []byte                    `gorm:"column:email_fingerprint"`
	Purpose              Purpose                   `gorm:"column:purpose"`
	CodeDigest           []byte                    `gorm:"column:code_digest"`
	State                State                     `gorm:"column:state"`
	ValidForSeconds      uint32                    `gorm:"column:valid_for_seconds"`
	MaxAttempts          uint32                    `gorm:"column:max_attempts"`
	FailedAttempts       uint32                    `gorm:"column:failed_attempts"`
	LatestDeliveryStatus deliveryv1.DeliveryStatus `gorm:"column:latest_delivery_status"`
	LatestSequence       uint64                    `gorm:"column:latest_sequence"`
	ActiveAt             *time.Time                `gorm:"column:active_at"`
	ExpiresAt            *time.Time                `gorm:"column:expires_at"`
	ConsumedAt           *time.Time                `gorm:"column:consumed_at"`
	CreatedAt            time.Time                 `gorm:"column:created_at"`
	UpdatedAt            time.Time                 `gorm:"column:updated_at"`
}

func (Challenge) TableName() string { return "email_verification_challenges" }

type DeliveryEvent struct {
	EventID       string
	MessageID     string
	RequestID     string
	Status        deliveryv1.DeliveryStatus
	Sequence      uint64
	AttemptNumber uint32
	OccurredAt    time.Time
}

type EventDisposition int

const (
	EventAccepted EventDisposition = iota + 1
	EventDuplicate
	EventIgnoredStale
)

type DeliverySnapshot struct {
	MessageID      string
	RequestID      string
	Status         deliveryv1.DeliveryStatus
	LatestSequence uint64
	OccurredAt     time.Time
}

type VerificationEmail struct {
	RequestID       string
	Recipient       string
	Code            string
	Purpose         Purpose
	ValidForSeconds uint32
	CreatedAt       time.Time
}

type Mailer interface {
	SubmitVerification(ctx context.Context, email VerificationEmail) (DeliverySnapshot, error)
	GetByRequestID(ctx context.Context, requestID string) (DeliverySnapshot, error)
}
