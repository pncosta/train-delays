# Build 
FROM golang:1.25.9-alpine AS builder

# Allow HTTPS connections
RUN apk add --no-cache ca-certificates
WORKDIR /app

# go dependencies   
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main .

# Run
FROM alpine:latest
RUN apk add --no-cache tzdata
WORKDIR /root/
COPY --from=builder /app/main .
# Use the PORT environment variable
ENV PORT=8080
CMD ["./main"]