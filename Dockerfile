# ── Build stage ───────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

# Allow the build to proceed even if go.mod specifies a newer Go toolchain minimum
ENV GOTOOLCHAIN=auto

WORKDIR /app

# Leverage Docker layer caching: download deps before copying source
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the binary
# - CGO_ENABLED=0: safe because modernc.org/sqlite is a pure-Go driver
# - trimpath: strips local file paths from the binary (security + reproducibility)
# - -s -w: strip debug info and DWARF tables → ~30% smaller binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o server \
    ./cmd/server

# ── Final stage ───────────────────────────────────────────────────────────────
FROM alpine:latest

# Runtime dependencies: timezone data + CA certificates (for HTTPS outbound calls)
RUN apk add --no-cache tzdata ca-certificates

# Create a non-root user to run the application
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy the compiled binary
COPY --from=builder /app/server .

# Copy HTML templates required by Fiber's template engine at runtime
COPY --from=builder /app/internal/interfaces/http/templates ./internal/interfaces/http/templates

# Copy static assets (CSS, JS, images)
COPY --from=builder /app/static ./static

# Copy the logo/header image used in PDF reports
COPY --from=builder /app/open_tdm.jpeg ./open_tdm.jpeg

# Transfer ownership to the non-root user
RUN chown -R appuser:appgroup /app

# Drop root privileges
USER appuser

# Note: for SQLite persistence mount a volume at /app:
#   docker run -v ./data:/app -p 8080:8080 <image-name>
EXPOSE 8080

# Use ENTRYPOINT so Docker properly forwards SIGTERM for graceful shutdown
ENTRYPOINT ["./server"]
