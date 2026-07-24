# syntax=docker/dockerfile:1

# —— Frontend ——
FROM node:22-alpine AS web-build
WORKDIR /src/web
COPY apps/web/package.json apps/web/package-lock.json* ./
RUN npm install
COPY apps/web/ ./
RUN npm run build

# —— Backend ——
FROM golang:1.23-alpine AS server-build
WORKDIR /src/server
RUN apk add --no-cache git ca-certificates
COPY apps/server/go.mod apps/server/go.sum ./
RUN go mod download
COPY apps/server/ ./
COPY --from=web-build /src/web/dist ./webdist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/booknav ./cmd/server

# —— Runtime ——
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=server-build /out/booknav /booknav
ENV BOOKNAV_HTTP_ADDR=:8080 \
    BOOKNAV_DATA_DIR=/data \
    BOOKNAV_STATIC_DIR=/webdist \
    BOOKNAV_ENV=production
# static files are embedded path via STATIC_DIR — copy webdist beside binary
COPY --from=server-build /src/server/webdist /webdist
# SQL migrations (resolved as ./migrations or beside binary)
COPY --from=server-build /src/server/migrations /migrations
USER nonroot:nonroot
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/booknav"]
