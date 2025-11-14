# backend/Dockerfile
# syntax=docker/dockerfile:1

FROM golang:1.24.4-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -ldflags="-s -w" -o /server ./cmd/api

# ★ CA 証明書入りの distroless base
FROM gcr.io/distroless/base-debian12:nonroot
COPY --from=builder /server /server
ENV PORT=8080
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/server"]
