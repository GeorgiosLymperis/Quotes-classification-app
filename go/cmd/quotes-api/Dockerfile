# -------- Build stage --------
FROM golang:1.24-alpine AS builder

# Το module root ΕΙΝΑΙ το go/
WORKDIR /app/go

# Copy go.mod & go.sum
COPY go/go.mod go/go.sum ./
RUN go mod download

# Copy the rest of the Go source
COPY go/ .

# Build quotes-api
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o quotes-api ./cmd/quotes-api

# -------- Runtime stage --------
FROM gcr.io/distroless/base-debian12

WORKDIR /app
COPY --from=builder /app/go/quotes-api /app/quotes-api

EXPOSE 8080
CMD ["/app/quotes-api"]
