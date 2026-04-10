# Build stage
FROM golang:1.24-alpine AS builder

# Allow the build to proceed even if go.mod specifies a newer Go toolchain minimum
ENV GOTOOLCHAIN=auto

# Set the working directory
WORKDIR /app

# Copy the go.mod and go.sum files first to leverage Docker layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application
# CGO_ENABLED=0 is safe here because modernc.org/sqlite is a pure-Go SQLite driver
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/server

# Final stage - using alpine for a smaller image size
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache tzdata ca-certificates

WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/main .

# Copy HTML templates required by Fiber at runtime
COPY --from=builder /app/internal/interfaces/http/templates ./internal/interfaces/http/templates

# Note: table_tennis.db is stored in /app.
# Mount a volume here to persist the database across restarts:
#   docker run -v ./data:/app -p 8080:8080 <image-name>

# Expose the port the app runs on
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
