// Package session provides revocable server-side login sessions.
package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	sessionKeyPrefix = "auth:session:"
	userSetKeyPrefix = "auth:user-sessions:"
)

// ErrNotFound indicates that a session has expired or has been revoked.
var ErrNotFound = errors.New("session not found")

// Store defines the session operations required by authentication and user flows.
type Store interface {
	Create(ctx context.Context, sessionID, userID string, ttl time.Duration) error
	UserID(ctx context.Context, sessionID string) (string, error)
	Delete(ctx context.Context, sessionID string) error
	DeleteAll(ctx context.Context, userID string) error
}

// RedisStore stores sessions in Redis.
type RedisStore struct {
	client redis.Cmdable
}

// NewRedisStore creates a Redis-backed session store.
func NewRedisStore(client redis.Cmdable) *RedisStore {
	return &RedisStore{client: client}
}

// Create stores a session and adds it to the user's revocation index.
func (s *RedisStore) Create(ctx context.Context, sessionID, userID string, ttl time.Duration) error {
	if sessionID == "" || userID == "" {
		return errors.New("session ID and user ID are required")
	}
	if ttl <= 0 {
		return errors.New("session TTL must be positive")
	}

	_, err := s.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, sessionKey(sessionID), userID, ttl)
		pipe.SAdd(ctx, userSetKey(userID), sessionID)
		// The index may outlive an older session, but never the newest session
		// created by this call. Stale members are harmless and removed by DeleteAll.
		pipe.Expire(ctx, userSetKey(userID), ttl)
		return nil
	})
	return err
}

// UserID returns the user that owns a live session.
func (s *RedisStore) UserID(ctx context.Context, sessionID string) (string, error) {
	userID, err := s.client.Get(ctx, sessionKey(sessionID)).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return userID, nil
}

// Delete revokes one session.
func (s *RedisStore) Delete(ctx context.Context, sessionID string) error {
	userID, err := s.UserID(ctx, sessionID)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	_, err = s.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, sessionKey(sessionID))
		pipe.SRem(ctx, userSetKey(userID), sessionID)
		return nil
	})
	return err
}

// DeleteAll revokes every known session for a user.
func (s *RedisStore) DeleteAll(ctx context.Context, userID string) error {
	sessionIDs, err := s.client.SMembers(ctx, userSetKey(userID)).Result()
	if err != nil {
		return fmt.Errorf("list user sessions: %w", err)
	}

	keys := make([]string, 0, len(sessionIDs)+1)
	for _, sessionID := range sessionIDs {
		keys = append(keys, sessionKey(sessionID))
	}
	keys = append(keys, userSetKey(userID))

	if err = s.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("delete user sessions: %w", err)
	}
	return nil
}

func sessionKey(sessionID string) string {
	return sessionKeyPrefix + sessionID
}

func userSetKey(userID string) string {
	return userSetKeyPrefix + userID
}
