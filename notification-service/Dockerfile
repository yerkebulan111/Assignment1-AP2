FROM golang:1.25.5-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY . .

RUN go mod tidy
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o notification-service ./cmd/notification-service

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/notification-service .

CMD ["./notification-service"]