# Build stage
FROM golang:1.24-alpine AS builder

# Set the working directory
WORKDIR /app

# Copy the go.mod and go.sum files first to leverage Docker layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application securely without CGO
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/server

# Final stage - using alpine for a smaller image size
FROM alpine:latest

# Install runtime dependencies such as tzdata for timezones
RUN apk add --no-cache tzdata

WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/main .

# Copy HTML templates required by Fiber at runtime (from root path)
COPY --from=builder /app/internal/interfaces/http/templates ./internal/interfaces/http/templates

# Note: table_tennis.db is stored in this directory. 
# You may want to mount a volume here for the database to persist across container restarts.
# Example: docker run -v ./data:/app/data -p 8080:8080 <image-name>

# Expose the port the app runs on
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
