# ============================================================================
# Stage 1: Builder — compile static Go binary
# ============================================================================
FROM golang:1.24-trixie AS builder

WORKDIR /src

# Download dependencies first (layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Build static binary
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server .

# ============================================================================
# Stage 2: Runtime — distroless (no shell, non-root)
# ============================================================================
FROM gcr.io/distroless/base-debian13

WORKDIR /app

COPY --from=builder /app/server /app/server

ENV PORT=8000

EXPOSE 8000

USER nonroot

ENTRYPOINT ["/app/server"]
