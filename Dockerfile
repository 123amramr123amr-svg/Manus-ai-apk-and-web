# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS build
WORKDIR /src

# The Manus AI Go SDK is vendored, so no module downloads are required.
COPY go.mod ./
COPY third_party ./third_party
COPY cmd ./cmd
COPY internal ./internal
COPY web ./web

ARG APP_VERSION=docker
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/manusweb ./cmd/manusweb

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 app
COPY --from=build /out/manusweb /usr/local/bin/manusweb

USER app
ENV PORT=8080 \
    APP_VERSION=${APP_VERSION}
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
    CMD wget -q -O- http://127.0.0.1:8080/api/health || exit 1

ENTRYPOINT ["/usr/local/bin/manusweb"]
