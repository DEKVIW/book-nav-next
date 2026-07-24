#!/usr/bin/env bash
# Tag local build and push to Docker Hub.
# Run on a machine that already has the image (e.g. mini-PC after compose build).
#
#   export DOCKERHUB_USER=yilan666
#   export DOCKERHUB_TOKEN=...   # or password; prefer Access Token
#   export VERSION=0.1.0         # optional; tags both VERSION and latest
#   bash deploy/push-hub.sh
#
set -euo pipefail

USER_NAME="${DOCKERHUB_USER:-yilan666}"
REPO="${DOCKERHUB_REPO:-booknav-next}"
VERSION="${VERSION:-0.1.0}"
SRC_IMAGE="${SRC_IMAGE:-deploy-booknav:latest}"
DEST="${USER_NAME}/${REPO}"

if ! docker image inspect "$SRC_IMAGE" >/dev/null 2>&1; then
  echo "[push] missing local image: $SRC_IMAGE" >&2
  echo "[push] build first: docker compose -f deploy/docker-compose.prod.yml build --build-arg VERSION=${VERSION}" >&2
  exit 1
fi

if [[ -n "${DOCKERHUB_TOKEN:-}" ]]; then
  echo "$DOCKERHUB_TOKEN" | docker login -u "$USER_NAME" --password-stdin
elif [[ -n "${DOCKERHUB_PASSWORD:-}" ]]; then
  echo "$DOCKERHUB_PASSWORD" | docker login -u "$USER_NAME" --password-stdin
else
  echo "[push] DOCKERHUB_TOKEN (or DOCKERHUB_PASSWORD) not set; trying existing docker login..." >&2
  docker info >/dev/null
fi

echo "[push] tag ${SRC_IMAGE} -> ${DEST}:${VERSION} and ${DEST}:latest"
docker tag "$SRC_IMAGE" "${DEST}:${VERSION}"
docker tag "$SRC_IMAGE" "${DEST}:latest"

docker push "${DEST}:${VERSION}"
docker push "${DEST}:latest"

echo "[push] done: https://hub.docker.com/r/${DEST}"
docker images "${DEST}" --format "table {{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.Size}}\t{{.CreatedSince}}"
