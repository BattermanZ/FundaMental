# Stage 1: Build Go binary
FROM golang:1.24-alpine AS go-builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o server ./cmd/server/main.go

# Stage 2: Python environment
FROM python:3.13-slim AS python-builder

# Set working directory
WORKDIR /app

# Copy Python requirements
COPY scripts/requirements.txt .

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy Python scripts
COPY scripts/scrapers ./scrapers

# Stage 3: Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache sqlite-libs python3 py3-pip tzdata

# Copy Go binary
COPY --from=go-builder /app/server /usr/local/bin/server

# Copy Python environment
COPY --from=python-builder /usr/local/lib/python3.13/site-packages /usr/local/lib/python3.13/site-packages
COPY --from=python-builder /app/scrapers /app/scrapers

# Create necessary directories
RUN mkdir -p /app/database

# Set working directory
WORKDIR /app

# Set environment variables
ENV GIN_MODE=release
ENV TZ=Europe/Amsterdam

# Expose port
EXPOSE 5250

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/server"] 