package verification

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	ValidFor          time.Duration
	PendingTTL        time.Duration
	Cooldown          time.Duration
	MaxAttempts       uint32
	HMACSecret        string
	FingerprintSecret string
}

type Service struct {
	store          *Store
	mailer         Mailer
	redis          *redis.Client
	config         Config
	digestKey      []byte
	fingerprintKey []byte
	now            func() time.Time
}

func NewService(store *Store, mailer Mailer, redisClient *redis.Client, cfg Config) (*Service, error) {
	if store == nil || mailer == nil || redisClient == nil {
		return nil, fmt.Errorf("verification dependencies are required")
	}
	if cfg.ValidFor < time.Minute || cfg.ValidFor > 30*time.Minute || cfg.PendingTTL <= 0 || cfg.Cooldown <= 0 || cfg.MaxAttempts == 0 {
		return nil, fmt.Errorf("invalid verification policy")
	}
	if len(cfg.HMACSecret) < 32 || len(cfg.FingerprintSecret) < 32 || hmac.Equal([]byte(cfg.HMACSecret), []byte(cfg.FingerprintSecret)) {
		return nil, fmt.Errorf("verification secrets must be distinct and at least 32 characters")
	}
	return &Service{
		store: store, mailer: mailer, redis: redisClient, config: cfg,
		digestKey: []byte(cfg.HMACSecret), fingerprintKey: []byte(cfg.FingerprintSecret), now: time.Now,
	}, nil
}

func (s *Service) Send(ctx context.Context, requestID, email string, purpose Purpose) (string, error) {
	if requestID == "" {
		requestID = uuid.NewString()
	} else if _, err := uuid.Parse(requestID); err != nil {
		return "", fmt.Errorf("%w: must be a UUID", ErrInvalidRequestID)
	}
	if existing, findErr := s.store.GetByRequestID(ctx, requestID); findErr == nil {
		if !hmac.Equal(existing.EmailFingerprint, s.fingerprint(normalizeEmail(email))) || existing.Purpose != purpose {
			return requestID, ErrChallengeConflict
		}
		snapshot, getErr := s.mailer.GetByRequestID(ctx, requestID)
		if getErr != nil {
			return requestID, fmt.Errorf("%w: %v", ErrDeliveryNotAccepted, getErr)
		}
		if applyErr := s.store.ApplySnapshot(ctx, snapshot, s.config.ValidFor); applyErr != nil {
			return requestID, applyErr
		}
		return requestID, nil
	} else if findErr != nil && findErr != ErrChallengeNotFound {
		return requestID, findErr
	}
	normalized := normalizeEmail(email)
	fingerprint := s.fingerprint(normalized)
	cooldownKey := "verification:cooldown:" + string(purpose) + ":" + hex.EncodeToString(fingerprint)
	set, err := s.redis.SetNX(ctx, cooldownKey, requestID, s.config.Cooldown).Result()
	if err != nil {
		return "", fmt.Errorf("set verification cooldown: %w", err)
	}
	if !set {
		return "", ErrCooldown
	}

	code, err := generateCode()
	if err != nil {
		_ = s.redis.Del(ctx, cooldownKey).Err()
		return "", err
	}
	now := s.now().UTC()
	challenge := &Challenge{
		RequestID: requestID, EmailFingerprint: fingerprint, Purpose: purpose,
		CodeDigest: s.digest(requestID, code), State: StatePendingDispatch,
		ValidForSeconds: uint32(s.config.ValidFor / time.Second), MaxAttempts: s.config.MaxAttempts,
		CreatedAt: now, UpdatedAt: now,
	}
	if err = s.store.CreatePending(ctx, challenge); err != nil {
		_ = s.redis.Del(ctx, cooldownKey).Err()
		return "", fmt.Errorf("create verification challenge: %w", err)
	}

	emailRequest := VerificationEmail{
		RequestID: requestID, Recipient: normalized, Code: code, Purpose: purpose,
		ValidForSeconds: challenge.ValidForSeconds, CreatedAt: now,
	}
	snapshot, err := s.mailer.SubmitVerification(ctx, emailRequest)
	if err != nil {
		if !errors.Is(err, ErrSubmissionUnknown) {
			_ = s.store.MarkDeliveryFailed(ctx, requestID)
			return requestID, ErrDeliveryNotAccepted
		}
		// The same request object and key make a transport-level retry safe. If that
		// outcome is also unknown, the status query converges on the original message.
		snapshot, err = s.mailer.SubmitVerification(ctx, emailRequest)
		if err != nil {
			snapshot, err = s.mailer.GetByRequestID(ctx, requestID)
		}
		if err != nil {
			return requestID, fmt.Errorf("%w: %v", ErrDeliveryNotAccepted, err)
		}
	}
	if err = snapshot.validate(); err != nil {
		return requestID, err
	}
	if err = s.store.ApplySnapshot(ctx, snapshot, s.config.ValidFor); err != nil {
		return requestID, fmt.Errorf("save mail acceptance: %w", err)
	}
	return requestID, nil
}

func (s *Service) VerifyAndConsume(ctx context.Context, email string, purpose Purpose, code string) error {
	if len(code) != 6 {
		return ErrCodeMismatch
	}
	return s.store.VerifyAndConsume(ctx, s.fingerprint(normalizeEmail(email)), purpose, code, s.digest)
}

func (s *Service) ApplyDeliveryEvent(ctx context.Context, event DeliveryEvent) (EventDisposition, error) {
	return s.store.ApplyEvent(ctx, event, s.config.ValidFor)
}

func (s *Service) Reconcile(ctx context.Context) error {
	items, err := s.store.PendingForReconciliation(ctx, s.now().Add(-s.config.PendingTTL), 100)
	if err != nil {
		return err
	}
	for _, item := range items {
		snapshot, getErr := s.mailer.GetByRequestID(ctx, item.RequestID)
		if getErr != nil {
			continue
		}
		_ = s.store.ApplySnapshot(ctx, snapshot, s.config.ValidFor)
	}
	return nil
}

func (s *Service) fingerprint(email string) []byte {
	mac := hmac.New(sha256.New, s.fingerprintKey)
	_, _ = mac.Write([]byte(email))
	return mac.Sum(nil)
}

func (s *Service) digest(requestID, code string) []byte {
	mac := hmac.New(sha256.New, s.digestKey)
	_, _ = mac.Write([]byte(requestID))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(code))
	return mac.Sum(nil)
}

func normalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func generateCode() (string, error) {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate verification code: %w", err)
	}
	return fmt.Sprintf("%06d", binary.BigEndian.Uint64(raw[:])%1_000_000), nil
}

func equalDigest(expected, actual []byte) bool {
	return len(expected) == len(actual) && subtle.ConstantTimeCompare(expected, actual) == 1
}
