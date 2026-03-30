# FROM golang:1.25.0-alpine AS builder
# WORKDIR /app

# COPY go.mod go.sum .
# RUN go mod download

# COPY ./web-server .
# RUN go build -o main .

# FROM alpine:latest
# WORKDIR /root/
# COPY --from=builder /app/main .
# # Use the PORT environment variable
# ENV PORT=8080
# CMD ["./main"]

# Build 
FROM golang:1.25.0-alpine AS builder

# Allow HTTPS connections
RUN apk add --no-cache ca-certificates
WORKDIR /app

COPY . .
RUN go build -o main .

# Run
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/main .
# Use the PORT environment variable
ENV PORT=8080
CMD ["./main"]