package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestStripIdentityHeaders_RemovesClientSuppliedUserID(t *testing.T) {
	var seen []string
	h := StripIdentityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Values(UserIDHeader)
	}))

	req := httptest.NewRequest(http.MethodGet, "/orders/1", nil)
	req.Header.Add("x-user-id", "admin") // lowercase: header names are case-insensitive
	req.Header.Add(UserIDHeader, "user-7")
	h.ServeHTTP(httptest.NewRecorder(), req)

	if len(seen) != 0 {
		t.Fatalf("backend saw %s = %q, want none", UserIDHeader, seen)
	}
}

func TestJWTAuth_ForwardsVerifiedIdentityNotToken(t *testing.T) {
	token := signedJWT(t, jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   "user-42",
		Issuer:    testJWTIssuer,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})

	var gotUserID, gotAuthorization string
	backend := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = r.Header.Get(UserIDHeader)
		gotAuthorization = r.Header.Get("Authorization")
	})
	// Same order as the gateway: strip first, then authenticate.
	h := StripIdentityHeaders(JWTAuth([]byte(testJWTSecret), testJWTIssuer)(backend))

	req := httptest.NewRequest(http.MethodGet, "/orders/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(UserIDHeader, "admin") // spoof attempt
	h.ServeHTTP(httptest.NewRecorder(), req)

	if gotUserID != "user-42" {
		t.Fatalf("backend saw %s = %q, want %q (from the verified token)", UserIDHeader, gotUserID, "user-42")
	}
	if gotAuthorization != "" {
		t.Fatalf("backend saw Authorization = %q, want it removed", gotAuthorization)
	}
}
