# --- Build Stage ---
FROM golang:1.26-alpine AS builder

# Sets the internal working directory to /app
WORKDIR /app

# Copy dependency files first (optimizes Docker layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project directory tree into /app
COPY . .

# Compile the gateway binary directly from its path inside /app/cmd/gateway
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/gateway ./cmd/gateway
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/admin ./cmd/admin

# --- Execution Stage ---
FROM alpine:3.19

WORKDIR /app

# Copy built binary from the builder stage
COPY --from=builder /app/gateway .
COPY --from=builder /app/admin .

EXPOSE 8080

CMD ["./gateway"]