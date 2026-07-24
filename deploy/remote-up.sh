#!/usr/bin/env bash
set -euo pipefail
ROOT="/data/app/my-service-test/book-nav"
cd "$ROOT"

if [[ ! -f .env ]]; then
  if [[ -f deploy/.env.docker.example ]]; then
    cp deploy/.env.docker.example .env
    echo "[deploy] created .env from deploy/.env.docker.example"
  else
    echo "[deploy] missing .env" >&2
    exit 1
  fi
fi

mkdir -p data
# distroless nonroot uid 65532 needs write on /data
chown -R 65532:65532 data 2>/dev/null || true
chmod -R u+rwX data 2>/dev/null || true

echo "[deploy] docker compose build & up ..."
if docker compose version >/dev/null 2>&1; then
  docker compose -f deploy/docker-compose.prod.yml up -d --build
else
  docker-compose -f deploy/docker-compose.prod.yml up -d --build
fi

echo "[deploy] status:"
docker ps --filter name=booknav
echo
HOST_IP=$(hostname -I 2>/dev/null | awk '{print $1}')
echo "[deploy] open: http://${HOST_IP:-192.168.8.117}:8988"
echo "[deploy] logs: docker logs -f booknav"