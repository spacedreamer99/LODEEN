# syntax=docker/dockerfile:1.7

FROM golang:1.27-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/

ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags "-s -w \
      -X github.com/spacedreamer99/lodeen/internal/shared/version.Version=${VERSION} \
      -X github.com/spacedreamer99/lodeen/internal/shared/version.Commit=${COMMIT} \
      -X github.com/spacedreamer99/lodeen/internal/shared/version.Date=${DATE}" \
    -o /out/lodeen-server ./cmd/server

FROM scratch
COPY --from=build /out/lodeen-server /lodeen-server
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

USER 65534:65534
EXPOSE 7777 9091

ENV LODEEN_ENV=prod \
    LODEEN_SERVER_TCP_ADDR=:7777 \
    LODEEN_SERVER_ADMIN_ADDR=:9091


ENTRYPOINT ["/lodeen-server"]
