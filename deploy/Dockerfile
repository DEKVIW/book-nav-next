# syntax=docker/dockerfile:1

# VERSION is baked into the binary (About page /api/v1/version).
# Local/dev default keeps -dev; Hub/prod builds should pass e.g. --build-arg VERSION=0.1.0
ARG VERSION=0.1.0-dev

# —— Frontend ——
FROM node:22-alpine AS web-build
WORKDIR /src/web
COPY apps/web/package.json apps/web/package-lock.json* ./
RUN npm install
COPY apps/web/ ./
RUN npm run build

# —— Backend ——
FROM golang:1.23-alpine AS server-build
ARG VERSION=0.1.0-dev
WORKDIR /src/server
RUN apk add --no-cache git ca-certificates
COPY apps/server/go.mod apps/server/go.sum ./
RUN go mod download
COPY apps/server/ ./
COPY --from=web-build /src/web/dist ./webdist
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X github.com/booknav/book-nav/apps/server/internal/pkg/version.Version=${VERSION}" \
    -o /out/booknav ./cmd/server

# —— Runtime ——
# Alpine (Docker Hub) instead of gcr.io/distroless — many VPS block gcr.
# UID 65532 matches prior nonroot convention for data volume ownership.
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -g 65532 -S nonroot \
    && adduser -u 65532 -S -G nonroot -H -D nonroot
WORKDIR /
COPY --from=server-build /out/booknav /booknav
ENV BOOKNAV_HTTP_ADDR=:8080 \
    BOOKNAV_DATA_DIR=/data \
    BOOKNAV_STATIC_DIR=/webdist \
    BOOKNAV_ENV=production
COPY --from=server-build /src/server/webdist /webdist
COPY --from=server-build /src/server/migrations /migrations
USER nonroot:nonroot
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/booknav"]
