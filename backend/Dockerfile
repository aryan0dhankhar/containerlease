# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .

RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o bin/server ./cmd/server

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk --no-cache add ca-certificates curl

COPY --from=builder /app/bin/server .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./server"]
