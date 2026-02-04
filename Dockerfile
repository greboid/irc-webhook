FROM docker.io/library/golang:1.26rc3 AS builder

WORKDIR /app
COPY . /app
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -trimpath -ldflags=-buildid= -o main ./cmd/webhook
RUN mkdir /data

FROM ghcr.io/greboid/dockerfiles/base

COPY --from=builder /app/main /irc-webhook
COPY --from=builder --chown=65532:65532 /data /data
CMD ["/irc-webhook"]
