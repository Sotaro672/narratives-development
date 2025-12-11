# backend/Dockerfile
# syntax=docker/dockerfile:1

FROM golang:1.24.4-alpine AS builder
WORKDIR /src

# Go modules
COPY go.mod go.sum ./
RUN go mod download

# App sources
COPY . .

# Build static binary for Linux/amd64
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -ldflags="-s -w" -o /server ./cmd/api

# ─────────────────────────────────────────────
# Runtime image (distroless, non-root)
# ─────────────────────────────────────────────
FROM gcr.io/distroless/base-debian12:nonroot

COPY --from=builder /server /server

# Default envs (本番では Cloud Run 側で上書き)
ENV PORT=8080 \
    ARWEAVE_BASE_URL="" 

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/server"]
