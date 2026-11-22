# API Sentinel

API Sentinel is a Go API gateway and reverse proxy.

## Running the gateway

The gateway uses API-key authentication by default. It requires its API key
from the `API_SENTINEL_API_KEY` environment variable. It must not be stored in
configuration files or source code.

```bash
API_SENTINEL_API_KEY='replace-with-a-secret' go run ./cmd/gateway
```

Clients must send the key in the `X-API-Key` header. Requests with a missing or
invalid key receive HTTP 401 and are not forwarded to an upstream service.

To use JWT authentication instead, select JWT mode and provide the signing
secret and expected issuer:

```bash
API_SENTINEL_AUTH_MODE=jwt \
API_SENTINEL_JWT_SECRET='replace-with-a-secret' \
API_SENTINEL_JWT_ISSUER='api-sentinel' \
go run ./cmd/gateway
```

JWT clients must send a signed HS256 token in the `Authorization` header using
the `Bearer` scheme. Tokens must contain a non-empty `sub`, the configured `iss`,
and a valid `exp` claim.

The gateway routes are configured in
[`configs/gateway.json`](./configs/gateway.json).

## Tests

```bash
go test ./...
```
