package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testJWTSecret = "test-jwt-secret"
	testJWTIssuer = "test-issuer"
)

func TestJWTAuth_ValidTokenAndSubjectContext(t *testing.T) {
	token := signedJWT(t, jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   "client-123",
		Issuer:    testJWTIssuer,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	})
	var subject string
	h := JWTAuth([]byte(testJWTSecret), testJWTIssuer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subject = AuthenticatedSubject(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := serveJWTRequest(h, "Bearer "+token)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if subject != "client-123" {
		t.Fatalf("subject = %q, want %q", subject, "client-123")
	}
}

func TestJWTAuth_MissingAuthorization(t *testing.T) {
	testJWTAuthRejects(t, "")
}

func TestJWTAuth_MalformedBearerHeader(t *testing.T) {
	testJWTAuthRejects(t, "Basic sample")
	testJWTAuthRejects(t, "Bearer")
	testJWTAuthRejects(t, "Bearer sample.jwt.value extra")
}

func TestJWTAuth_InvalidSignature(t *testing.T) {
	token := signedJWTWithSecret(t, jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   "client-123",
		Issuer:    testJWTIssuer,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}, "wrong-secret")
	testJWTAuthRejects(t, "Bearer "+token)
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	token := signedJWT(t, jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   "client-123",
		Issuer:    testJWTIssuer,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
	})
	testJWTAuthRejects(t, "Bearer "+token)
}

func TestJWTAuth_FutureNotBefore(t *testing.T) {
	token := signedJWT(t, jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   "client-123",
		Issuer:    testJWTIssuer,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		NotBefore: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})
	testJWTAuthRejects(t, "Bearer "+token)
}

func TestJWTAuth_MissingSubject(t *testing.T) {
	token := signedJWT(t, jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    testJWTIssuer,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})
	testJWTAuthRejects(t, "Bearer "+token)
}

func TestJWTAuth_MissingExpiration(t *testing.T) {
	token := signedJWT(t, jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject: "client-123",
		Issuer:  testJWTIssuer,
	})
	testJWTAuthRejects(t, "Bearer "+token)
}

func TestJWTAuth_IncorrectIssuer(t *testing.T) {
	token := signedJWT(t, jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   "client-123",
		Issuer:    "wrong-issuer",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})
	testJWTAuthRejects(t, "Bearer "+token)
}

func TestJWTAuth_UnsupportedSigningAlgorithm(t *testing.T) {
	token := signedJWT(t, jwt.SigningMethodHS384, jwt.RegisteredClaims{
		Subject:   "client-123",
		Issuer:    testJWTIssuer,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})
	testJWTAuthRejects(t, "Bearer "+token)
}

func TestNewJWTAuthFromEnv_MissingSecret(t *testing.T) {
	t.Setenv(JWTSecretEnv, "")
	t.Setenv(JWTIssuerEnv, testJWTIssuer)

	if _, err := NewJWTAuthFromEnv(); err == nil {
		t.Fatal("NewJWTAuthFromEnv() returned nil error")
	}
}

func TestNewJWTAuthFromEnv_ValidConfiguration(t *testing.T) {
	t.Setenv(JWTSecretEnv, strings.Repeat("s", MinJWTSecretBytes))
	t.Setenv(JWTIssuerEnv, testJWTIssuer)

	if _, err := NewJWTAuthFromEnv(); err != nil {
		t.Fatalf("NewJWTAuthFromEnv() error = %v", err)
	}
}

func TestNewJWTAuthFromEnv_ShortSecret(t *testing.T) {
	t.Setenv(JWTSecretEnv, strings.Repeat("s", MinJWTSecretBytes-1))
	t.Setenv(JWTIssuerEnv, testJWTIssuer)

	if _, err := NewJWTAuthFromEnv(); err == nil {
		t.Fatal("NewJWTAuthFromEnv() accepted a secret shorter than the minimum")
	}
}

func TestJWTAuth_ClockLeeway(t *testing.T) {
	tests := []struct {
		name      string
		expiredBy time.Duration
		wantCode  int
	}{
		{"expired 10s ago is accepted (within leeway)", 10 * time.Second, http.StatusNoContent},
		{"expired 2m ago is rejected (beyond leeway)", 2 * time.Minute, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := signedJWT(t, jwt.SigningMethodHS256, jwt.RegisteredClaims{
				Subject:   "client-123",
				Issuer:    testJWTIssuer,
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-tt.expiredBy)),
			})
			h := JWTAuth([]byte(testJWTSecret), testJWTIssuer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))

			if rec := serveJWTRequest(h, "Bearer "+token); rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

func testJWTAuthRejects(t *testing.T, authorization string) {
	t.Helper()
	called := false
	h := JWTAuth([]byte(testJWTSecret), testJWTIssuer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	rec := serveJWTRequest(h, authorization)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if called {
		t.Fatal("downstream handler was called for an invalid token")
	}
	if strings.Contains(rec.Body.String(), testJWTSecret) {
		t.Fatal("response contains the JWT secret")
	}
}

func serveJWTRequest(h http.Handler, authorization string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func signedJWT(t *testing.T, method jwt.SigningMethod, claims jwt.RegisteredClaims) string {
	return signedJWTWithSecret(t, method, claims, testJWTSecret)
}

func signedJWTWithSecret(t *testing.T, method jwt.SigningMethod, claims jwt.RegisteredClaims, secret string) string {
	t.Helper()
	token, err := jwt.NewWithClaims(method, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return token
}
