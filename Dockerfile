# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /app/dinodb-server .

# Final stage
FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/dinodb-server /app/dinodb-server
COPY --from=builder /app/image /app/image

RUN chmod +x /app/dinodb-server

EXPOSE 8080
VOLUME ["/data"]

CMD ["/app/dinodb-server"]