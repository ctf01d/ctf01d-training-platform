ARG GO_VERSION=1.26
ARG NODE_VERSION=22

FROM golang:${GO_VERSION}-alpine AS go-builder
RUN apk add --no-cache git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags "-X main.version=${VERSION}" -o /ctf01d-server ./cmd/server

FROM node:${NODE_VERSION}-alpine AS web-builder
WORKDIR /web
COPY web/package.json web/package-lock.json* ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata su-exec
RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=go-builder /ctf01d-server ./ctf01d-server
COPY --from=go-builder /src/migrations ./migrations
COPY --from=web-builder /web/dist ./web/dist
RUN mkdir -p /data/storage /web-dist && chown -R app:app /data /web-dist /app
USER app
ENV RUN_MIGRATIONS=false
ENV APP_ENV=production
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8080/healthz || exit 1
ENTRYPOINT ["./ctf01d-server"]
