FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY notification-service/go.mod notification-service/go.sum ./notification-service/
COPY shared/events/ ./shared/events/

RUN cd notification-service && GOWORK=off go mod download

COPY notification-service/ ./notification-service/

RUN cd notification-service && GOWORK=off CGO_ENABLED=0 go build -o /bin/server ./cmd/main.go

FROM alpine:3.21

RUN apk add --no-cache ca-certificates \
    && addgroup -S -g 10001 app \
    && adduser -S -u 10001 -G app app

COPY --from=builder --chown=app:app /bin/server /bin/server
COPY --from=builder --chown=app:app /app/notification-service/migrations /migrations

USER app

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8081/_info || exit 1

EXPOSE 8081

CMD ["/bin/server"]
