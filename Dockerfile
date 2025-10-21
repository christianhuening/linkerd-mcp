# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o linkerd-mcp .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

# Create a non-root user and group
RUN addgroup -g 65532 -S nonroot && adduser -u 65532 -S nonroot -G nonroot

# Use /app as the working directory (accessible by all users)
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/linkerd-mcp .

# Make the binary executable and readable by all users
RUN chmod 755 /app/linkerd-mcp

# Expose port (if using HTTP transport for MCP)
EXPOSE 8080

# Run the binary
CMD ["./linkerd-mcp"]
