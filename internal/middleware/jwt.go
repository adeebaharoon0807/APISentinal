package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	AuthModeEnv    = "API_SENTINEL_AUTH_MODE"
	AuthModeAPIKey = "api_key"
	AuthModeJWT    = "jwt"
	JWTSecretEnv   = "API_SENTINEL_JWT_SECRET"
	JWTIssuerEnv   = "API_SENTINEL_JWT_ISSUER"

	// MinJWTSecretBytes matches HS256's 256-bit output; shorter secrets can be brute-forced offline.
	MinJWTSecretBytes = 32
	// JWTClockLeeway tolerates small clock differences between the token issuer and the gateway.
	JWTClockLeeway = 30 * time.Second
)

type jwtSubjectContextKey struct{}

// NewJWTAuthFromEnv creates HS256 JWT authentication using startup
// configuration. Secrets and tokens are never logged or returned to clients.
func NewJWTAuthFromEnv() (Middleware, error) {
	secret, ok := os.LookupEnv(JWTSecretEnv)
	if !ok || strings.TrimSpace(secret) == "" {
		return nil, errors.New("JWT authentication requires " + JWTSecretEnv)
	}
	if len(secret) < MinJWTSecretBytes {
		return nil, fmt.Errorf("%s must be at least %d bytes (generate one with: openssl rand -base64 32)", JWTSecretEnv, MinJWTSecretBytes)
	}

	issuer, ok := os.LookupEnv(JWTIssuerEnv)
	if !ok || strings.TrimSpace(issuer) == "" {
		return nil, errors.New("JWT authentication requires " + JWTIssuerEnv)
	}

	return JWTAuth([]byte(secret), issuer), nil
}

func JWTAuth(secret []byte, expectedIssuer string) Middleware {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(expectedIssuer),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(JWTClockLeeway),
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fields := strings.Fields(r.Header.Get("Authorization"))
			if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
				writeUnauthorized(w)
				return
			}

			claims := &jwt.RegisteredClaims{}
			token, err := parser.ParseWithClaims(fields[1], claims, func(token *jwt.Token) (any, error) {
				if token.Method != jwt.SigningMethodHS256 {
					return nil, errors.New("unexpected signing method")
				}
				return secret, nil
			})
			if err != nil || token == nil || !token.Valid || claims.Subject == "" {
				writeUnauthorized(w)
				return
			}

			// Backends get the verified identity, never the raw token.
			r.Header.Del("Authorization")
			r.Header.Set(UserIDHeader, claims.Subject)

			ctx := context.WithValue(r.Context(), jwtSubjectContextKey{}, claims.Subject)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AuthenticatedSubject(ctx context.Context) string {
	subject, _ := ctx.Value(jwtSubjectContextKey{}).(string)
	return subject
}

func writeUnauthorized(w http.ResponseWriter) {
	http.Error(w, "unauthorized", http.StatusUnauthorized)
}
