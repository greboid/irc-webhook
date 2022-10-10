FROM ghcr.io/greboid/dockerfiles/golang as builder

ENV USER=appuser
ENV UID=10001

RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

WORKDIR /app
COPY . /app
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -trimpath -ldflags=-buildid= -o main ./cmd/webhook


FROM ghcr.io/greboid/dockerfiles/base

COPY --from=builder /app/main /irc-webhook
USER appuser:appuser
CMD ["/irc-webhook"]
