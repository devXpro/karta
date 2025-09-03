# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies for CGO (SQLite needs gcc)
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o karta cmd/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests, sqlite and timezone data
RUN apk --no-cache add ca-certificates sqlite tzdata

# Set timezone to Europe/Warsaw (Poland, UTC+2)
ENV TZ=Europe/Warsaw
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/karta .

# Create directory for database
RUN mkdir -p /data

# Expose port (not really needed since we're using network_mode: service:surfshark)
EXPOSE 8080

# Run the application
CMD ["./karta"]
