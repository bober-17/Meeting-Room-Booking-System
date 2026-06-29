FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY notification-service/go.mod notification-service/go.sum ./notification-service/
COPY shared/events/ ./shared/events/

RUN cd notification-service && GOWORK=off go mod download

COPY notification-service/ ./notification-service/

RUN cd notification-service && GOWORK=off CGO_ENABLED=0 go build -o /notification-service ./cmd/main.go

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /notification-service .
COPY notification-service/migrations/ ./migrations/

EXPOSE 8081
CMD ["./notification-service"]
