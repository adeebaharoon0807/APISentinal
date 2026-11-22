# ---- Stage 1: build ----
FROM golang:1.25-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG TARGET
RUN CGO_ENABLED=0 go build -o /out/app ./cmd/${TARGET}

# ---- Stage 2: run ----
FROM alpine:3.20
COPY --from=build /out/app /app
ENTRYPOINT ["/app"]
