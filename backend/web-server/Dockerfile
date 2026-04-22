# Build 
FROM golang:1.25.9-alpine AS builder

# Allow HTTPS connections
RUN apk add --no-cache ca-certificates
WORKDIR /app

# 1. Copy the workspace and module files first
COPY go.work ./
COPY shared/go.mod ./shared/
COPY web-server/go.mod ./web-server/
COPY scraper/go.mod ./scraper/

# 2. Download dependencies
RUN go mod download

# 3. Copy the actual source code
COPY shared/ ./shared/
COPY web-server/ ./web-server/

# 4. Build the specific binary
WORKDIR /app/web-server
RUN go build -o /main .


# Run
FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /main /main 
# Use the PORT environment variable
ENV PORT=8080
CMD ["./main"]