# Stage 1: Build the binary
FROM golang:1.25.0-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o main .

# Stage 2: Run the binary in a tiny image
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/main .
# Use the PORT environment variable
ENV PORT=8080
CMD ["./main"]