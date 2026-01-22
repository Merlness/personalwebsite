# Build stage
FROM golang:1.23-alpine AS builder

# Install Node.js for Tailwind
RUN apk add --no-cache nodejs npm

WORKDIR /app

# Copy dependency definitions
COPY go.mod go.sum ./
COPY package.json package-lock.json ./

# Install dependencies
RUN go mod download
RUN npm ci

# Copy source code
COPY . .

# Generate Templ code
RUN go run github.com/a-h/templ/cmd/templ@latest generate

# Build CSS
RUN npm run build-css -- --minify

# Build Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/server/main.go

# Run warmup to pre-generate image cache
# We set CACHE_DIR to a local directory that we will copy to the final image
ENV CACHE_DIR=/app/image_cache
RUN mkdir -p ${CACHE_DIR}
RUN go run ./cmd/warmup/main.go

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy binary
COPY --from=builder /app/main .

# Copy content and assets
COPY --from=builder /app/content ./content
COPY --from=builder /app/internal/assets ./internal/assets

# Copy pre-warmed image cache
COPY --from=builder /app/image_cache ./image_cache

# Environment variables
ENV PORT=8080
ENV CACHE_DIR=/app/image_cache
ENV GIN_MODE=release

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]
