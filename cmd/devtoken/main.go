// Command devtoken mints JWTs for local testing of the gateway's JWT mode.
// It reads the same secret and issuer as the gateway, so tokens never need
// to be created on a third-party website.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/adeeba-bipl/api-sentinel/internal/middleware"
)

func main() {
	sub := flag.String("sub", "user-42", "subject (user ID) to put in the token")
	ttl := flag.Duration("ttl", 15*time.Minute, "token lifetime; a negative value creates an already-expired token")
	algNone := flag.Bool("alg-none", false, "create an UNSIGNED alg=none token (attack testing only)")
	flag.Parse()

	secret := os.Getenv(middleware.JWTSecretEnv)
	issuer := os.Getenv(middleware.JWTIssuerEnv)
	if secret == "" || issuer == "" {
		log.Fatalf("set %s and %s (the same values the gateway uses)", middleware.JWTSecretEnv, middleware.JWTIssuerEnv)
	}

	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   *sub,
		Issuer:    issuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(*ttl)),
	}

	var token string
	var err error
	if *algNone {
		token, err = jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	} else {
		token, err = jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	}
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(token)
}
