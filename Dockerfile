# Runtime image only. GoReleaser provides the pre-built binary and extra_files.
# Before docker build: make generate && make frontend (goreleaser before hooks run make generate, make frontend).
# For local build: make generate && make frontend && make service && cp service/easypour-service . && cp service/config.yaml service/menu.yaml . && docker build -f Dockerfile .
# menu.yaml is an optional first-boot seed; the live menu is stored in /config/easypour.db.
FROM golang:1.25-alpine AS sqlmigrate
RUN go install github.com/rubenv/sql-migrate/sql-migrate@v1.8.1

FROM alpine:3.20
LABEL org.opencontainers.image.title=EasyPour
RUN apk add --no-cache ca-certificates
COPY --from=sqlmigrate /go/bin/sql-migrate /usr/bin/sql-migrate
EXPOSE 9654
COPY easypour-service /usr/bin/easypour-service
COPY service/config.yaml service/menu.yaml /config/
COPY frontend/dist /app/frontend
COPY database /var/app/database
COPY docker-entrypoint.sh /usr/local/bin/easypour-entrypoint.sh
RUN chmod +x /usr/local/bin/easypour-entrypoint.sh /usr/bin/easypour-service /usr/bin/sql-migrate
WORKDIR /app
ENV EASYPOUR_CONFIG_FILE=/config/config.yaml
ENV EASYPOUR_STATIC_DIR=/app/frontend
ENV DB_DRIVER=sqlite
ENV DB_PATH=/config/easypour.db
VOLUME /config
ENTRYPOINT ["/usr/local/bin/easypour-entrypoint.sh"]
