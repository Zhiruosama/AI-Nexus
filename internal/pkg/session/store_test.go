package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisStoreSessionLifecycle(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close Redis client: %v", err)
		}
	})

	store := NewRedisStore(client)
	ctx := context.Background()
	if err := store.Create(ctx, "session-1", "user-1", time.Hour); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := store.Create(ctx, "session-2", "user-1", time.Hour); err != nil {
		t.Fatalf("Create() second session error = %v", err)
	}

	userID, err := store.UserID(ctx, "session-1")
	if err != nil {
		t.Fatalf("UserID() error = %v", err)
	}
	if userID != "user-1" {
		t.Fatalf("UserID() = %q, want %q", userID, "user-1")
	}

	if err = store.Delete(ctx, "session-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err = store.UserID(ctx, "session-1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("UserID() after Delete error = %v, want ErrNotFound", err)
	}
	if userID, secondErr := store.UserID(ctx, "session-2"); secondErr != nil || userID != "user-1" {
		t.Fatalf("unrelated session after Delete = (%q, %v), want (%q, nil)", userID, secondErr, "user-1")
	}
	if err = store.Delete(ctx, "session-1"); err != nil {
		t.Fatalf("idempotent Delete() error = %v", err)
	}
}

func TestRedisStoreDeleteAllUserSessions(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close Redis client: %v", err)
		}
	})

	store := NewRedisStore(client)
	ctx := context.Background()
	for _, item := range []struct {
		sessionID string
		userID    string
	}{
		{sessionID: "user-1-phone", userID: "user-1"},
		{sessionID: "user-1-pc", userID: "user-1"},
		{sessionID: "user-2-pc", userID: "user-2"},
	} {
		if err := store.Create(ctx, item.sessionID, item.userID, time.Hour); err != nil {
			t.Fatalf("Create(%q) error = %v", item.sessionID, err)
		}
	}

	if err := store.DeleteAll(ctx, "user-1"); err != nil {
		t.Fatalf("DeleteAll() error = %v", err)
	}
	for _, sessionID := range []string{"user-1-phone", "user-1-pc"} {
		if _, err := store.UserID(ctx, sessionID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("UserID(%q) error = %v, want ErrNotFound", sessionID, err)
		}
	}
	if userID, err := store.UserID(ctx, "user-2-pc"); err != nil || userID != "user-2" {
		t.Fatalf("unrelated session = (%q, %v), want (%q, nil)", userID, err, "user-2")
	}
}

func TestRedisStoreSessionTTL(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close Redis client: %v", err)
		}
	})

	store := NewRedisStore(client)
	ctx := context.Background()
	if err := store.Create(ctx, "short-session", "user-1", time.Minute); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	server.FastForward(61 * time.Second)
	if _, err := store.UserID(ctx, "short-session"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired UserID() error = %v, want ErrNotFound", err)
	}
}
