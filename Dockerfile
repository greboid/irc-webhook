FROM ghcr.io/greboid/dockerfiles/golang@sha256:65e504b0cb4e5df85e2301f47cd3f231768d7b0d5aba59b1201e9c50fdf5e0ac as builder

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


FROM ghcr.io/greboid/dockerfiles/base@sha256:82873fbcddc94e3cf77fdfe36765391b6e6049701623a62c2a23248d2a42b1cf

COPY --from=builder /app/main /irc-webhook
USER appuser:appuser
CMD ["/irc-webhook"]
