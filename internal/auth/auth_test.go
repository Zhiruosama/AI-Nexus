package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Zhiruosama/ai_nexus/internal/pkg/session"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const testJWTSecret = "test-only-secret-with-sufficient-length"

func TestGenerateAndParseToken(t *testing.T) {
	t.Setenv("JWT_SECRET", testJWTSecret)

	generated, err := GenerateToken("user-123")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if generated.SessionID == "" {
		t.Fatal("GenerateToken() returned empty session ID")
	}

	claims, err := ParseToken(generated.Value)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if claims.Subject != "user-123" {
		t.Fatalf("claims.Subject = %q, want %q", claims.Subject, "user-123")
	}
	if claims.ID != generated.SessionID {
		t.Fatalf("claims.ID = %q, want %q", claims.ID, generated.SessionID)
	}
	if claims.Issuer != tokenIssuer {
		t.Fatalf("claims.Issuer = %q, want %q", claims.Issuer, tokenIssuer)
	}
	if claims.ExpiresAt == nil || time.Until(claims.ExpiresAt.Time) <= 0 {
		t.Fatal("token expiration is missing or not in the future")
	}
}

func TestParseTokenRejectsExpiredToken(t *testing.T) {
	t.Setenv("JWT_SECRET", testJWTSecret)

	token := signedToken(t, jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-123",
			ID:        "session-123",
			Issuer:    tokenIssuer,
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	})

	_, err := ParseToken(token)
	if !errors.Is(err, jwt.ErrTokenExpired) {
		t.Fatalf("ParseToken() error = %v, want ErrTokenExpired", err)
	}
}

func TestParseTokenRejectsUnexpectedAlgorithm(t *testing.T) {
	t.Setenv("JWT_SECRET", testJWTSecret)

	token := signedToken(t, jwt.SigningMethodHS384, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-123",
			ID:        "session-123",
			Issuer:    tokenIssuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	if _, err := ParseToken(token); err == nil {
		t.Fatal("ParseToken() accepted a token signed with HS384")
	}
}

func TestParseTokenRejectsTamperedSignature(t *testing.T) {
	t.Setenv("JWT_SECRET", testJWTSecret)

	generated, err := GenerateToken("user-123")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	signatureStart := strings.LastIndex(generated.Value, ".") + 1
	replacement := byte('x')
	if generated.Value[signatureStart] == replacement {
		replacement = 'y'
	}
	tampered := generated.Value[:signatureStart] + string(replacement) + generated.Value[signatureStart+1:]

	if _, err = ParseToken(tampered); err == nil {
		t.Fatal("ParseToken() accepted a token with a tampered signature")
	}
}

func TestMiddlewareAcceptsLiveSession(t *testing.T) {
	t.Setenv("JWT_SECRET", testJWTSecret)
	gin.SetMode(gin.TestMode)

	generated, err := GenerateToken("user-123")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	store := newFakeStore()
	store.sessions[generated.SessionID] = "user-123"

	var gotPrincipal *Principal
	router := gin.New()
	router.Use(Middleware(store))
	router.GET("/protected", func(c *gin.Context) {
		gotPrincipal, _ = CurrentPrincipal(c)
		c.Status(http.StatusNoContent)
	})

	response := performRequest(router, generated.Value)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if gotPrincipal == nil || gotPrincipal.UserID != "user-123" ||
		gotPrincipal.SessionID != generated.SessionID {
		t.Fatalf("principal = %#v, want authenticated user and session", gotPrincipal)
	}
}

func TestMiddlewareRejectsTokenImmediatelyAfterSessionDeletion(t *testing.T) {
	t.Setenv("JWT_SECRET", testJWTSecret)
	gin.SetMode(gin.TestMode)

	generated, err := GenerateToken("user-123")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	store := newFakeStore()
	if err = store.Create(context.Background(), generated.SessionID, "user-123", time.Hour); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	router := gin.New()
	router.Use(Middleware(store))
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	if response := performRequest(router, generated.Value); response.Code != http.StatusNoContent {
		t.Fatalf("status before revocation = %d, want %d", response.Code, http.StatusNoContent)
	}
	if err = store.Delete(context.Background(), generated.SessionID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if response := performRequest(router, generated.Value); response.Code != http.StatusUnauthorized {
		t.Fatalf("status after revocation = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestMiddlewareRejectsRevokedOrMismatchedSession(t *testing.T) {
	t.Setenv("JWT_SECRET", testJWTSecret)
	gin.SetMode(gin.TestMode)

	generated, err := GenerateToken("user-123")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	tests := []struct {
		name     string
		sessions map[string]string
	}{
		{name: "revoked session", sessions: map[string]string{}},
		{
			name:     "session belongs to another user",
			sessions: map[string]string{generated.SessionID: "user-456"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := newFakeStore()
			store.sessions = test.sessions
			handlerCalled := false
			router := gin.New()
			router.Use(Middleware(store))
			router.GET("/protected", func(c *gin.Context) {
				handlerCalled = true
				c.Status(http.StatusNoContent)
			})

			response := performRequest(router, generated.Value)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
			}
			if handlerCalled {
				t.Fatal("protected handler was called after authentication failed")
			}
		})
	}
}

func TestMiddlewareFailsClosedWhenSessionStoreIsUnavailable(t *testing.T) {
	t.Setenv("JWT_SECRET", testJWTSecret)
	gin.SetMode(gin.TestMode)

	generated, err := GenerateToken("user-123")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	store := newFakeStore()
	store.userIDErr = errors.New("redis unavailable")

	handlerCalled := false
	router := gin.New()
	router.Use(Middleware(store))
	router.GET("/protected", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusNoContent)
	})

	response := performRequest(router, generated.Value)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if handlerCalled {
		t.Fatal("protected handler was called while session store was unavailable")
	}
}

func TestMiddlewareRejectsMalformedAuthorization(t *testing.T) {
	t.Setenv("JWT_SECRET", testJWTSecret)
	gin.SetMode(gin.TestMode)

	handlerCalled := false
	router := gin.New()
	router.Use(Middleware(newFakeStore()))
	router.GET("/protected", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "not-a-bearer-token")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if handlerCalled {
		t.Fatal("protected handler was called with malformed authorization")
	}
}

func signedToken(t *testing.T, method jwt.SigningMethod, claims Claims) string {
	t.Helper()
	value, err := jwt.NewWithClaims(method, claims).SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return value
}

func performRequest(router http.Handler, token string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

type fakeStore struct {
	sessions  map[string]string
	userIDErr error
}

func newFakeStore() *fakeStore {
	return &fakeStore{sessions: make(map[string]string)}
}

func (s *fakeStore) Create(_ context.Context, sessionID, userID string, _ time.Duration) error {
	s.sessions[sessionID] = userID
	return nil
}

func (s *fakeStore) UserID(_ context.Context, sessionID string) (string, error) {
	if s.userIDErr != nil {
		return "", s.userIDErr
	}
	userID, ok := s.sessions[sessionID]
	if !ok {
		return "", session.ErrNotFound
	}
	return userID, nil
}

func (s *fakeStore) Delete(_ context.Context, sessionID string) error {
	delete(s.sessions, sessionID)
	return nil
}

func (s *fakeStore) DeleteAll(_ context.Context, userID string) error {
	for sessionID, sessionUserID := range s.sessions {
		if sessionUserID == userID {
			delete(s.sessions, sessionID)
		}
	}
	return nil
}
