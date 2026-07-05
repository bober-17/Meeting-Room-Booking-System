FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY booking-service/go.mod booking-service/go.sum ./
RUN go mod download

COPY booking-service/ .

RUN CGO_ENABLED=0 go build -o /bin/server ./cmd

FROM alpine:3.21

RUN apk add --no-cache ca-certificates \
    && addgroup -S -g 10001 app \
    && adduser -S -u 10001 -G app app

COPY --from=builder --chown=app:app /bin/server /bin/server
COPY --from=builder --chown=app:app /app/migrations /migrations

USER app

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/_info || exit 1

EXPOSE 8080

CMD ["/bin/server"]
