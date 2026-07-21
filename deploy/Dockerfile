# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS go-api
WORKDIR /src
COPY pkg/ pkg/
COPY apps/api/ apps/api/
WORKDIR /src/apps/api
ENV GOWORK=off
RUN CGO_ENABLED=0 go build -o /api ./cmd/api

FROM golang:1.25-alpine AS go-worker
WORKDIR /src
COPY pkg/ pkg/
COPY apps/worker/ apps/worker/
WORKDIR /src/apps/worker
ENV GOWORK=off
RUN CGO_ENABLED=0 go build -o /worker ./cmd/worker

FROM node:20-alpine AS web
WORKDIR /src
COPY apps/web/package.json apps/web/package-lock.json* ./apps/web/
WORKDIR /src/apps/web
RUN npm ci || npm install
COPY apps/web .
RUN npm run build

FROM nginx:1.27-alpine

ARG TARGETARCH
RUN apk add --no-cache bash curl ca-certificates \
  && curl -fsSL "https://github.com/golang-migrate/migrate/releases/download/v4.17.1/migrate.linux-${TARGETARCH}.tar.gz" \
    | tar xz -C /usr/local/bin \
  && chmod +x /usr/local/bin/migrate \
  && apk del curl

COPY --from=go-api /api /app/api
COPY --from=go-worker /worker /app/worker
COPY pkg/i18n/messages /app/pkg/i18n/messages
COPY --from=web /src/apps/web/dist /usr/share/nginx/html
COPY db/migrations /migrations
COPY deploy/nginx.all-in-one.conf /etc/nginx/conf.d/default.conf
COPY deploy/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh /app/api /app/worker

ENV HTTP_ADDR=127.0.0.1:8080
EXPOSE 3000
WORKDIR /app
ENTRYPOINT ["/entrypoint.sh"]
