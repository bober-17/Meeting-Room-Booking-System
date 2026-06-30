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

# HEALTHCHECK добавить в фазе 6 после реализации GET /health или аналогичного endpoint-а

EXPOSE 8081

CMD ["/bin/server"]
