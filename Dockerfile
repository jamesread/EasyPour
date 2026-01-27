# Runtime image only. GoReleaser provides the pre-built binary and extra_files.
# Before docker build: make generate && make frontend (goreleaser before hooks run make generate, make frontend).
# For local build: make generate && make frontend && make service && cp service/easypour-service . && cp service/config.yaml service/menu.yaml . && docker build -f Dockerfile .
FROM alpine:3.20
LABEL org.opencontainers.image.title=EasyPour
RUN apk add --no-cache ca-certificates
EXPOSE 9654
COPY easypour-service /usr/bin/easypour-service
COPY service/config.yaml service/menu.yaml /app/
COPY frontend/dist /app/frontend
WORKDIR /app
ENV EASYPOUR_CONFIG_FILE=/app/config.yaml
ENV EASYPOUR_STATIC_DIR=/app/frontend
ENTRYPOINT ["/usr/bin/easypour-service"]
