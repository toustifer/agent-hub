#!/usr/bin/env bash
# Build release artifacts for agent-hub (Linux server binary + frontend static).
# Usage: ./scripts/build-release.sh [version]
# Example: ./scripts/build-release.sh v0.2.0
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  if [[ -f VERSION ]]; then
    VERSION="v$(tr -d '[:space:]' < VERSION)"
  else
    VERSION="v0.0.0-dev"
  fi
fi
# ensure leading v
[[ "$VERSION" == v* ]] || VERSION="v$VERSION"

COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
LDFLAGS="-s -w -X github.com/stifer/agent-hub/internal/version.Version=${VERSION} -X github.com/stifer/agent-hub/internal/version.Commit=${COMMIT} -X github.com/stifer/agent-hub/internal/version.BuildTime=${BUILD_TIME}"

OUT="dist/release/${VERSION}"
rm -rf "$OUT"
mkdir -p "$OUT"

echo "==> Building hub-server linux/amd64 (${VERSION} ${COMMIT})"
export CGO_ENABLED=0
export GOOS=linux
export GOARCH=amd64
go build -ldflags "$LDFLAGS" -o "$OUT/hub-server" ./cmd/hub

echo "==> Building frontend"
if [[ -d frontend ]]; then
  (cd frontend && npm ci --no-audit --no-fund 2>/dev/null || npm install --no-audit --no-fund)
  (cd frontend && npx vite build)
  mkdir -p "$OUT/static"
  cp -R frontend/dist/* "$OUT/static/"
  # public markdown docs for AI paste
  for f in agent-setup.md agentflow-setup.md mcp.md; do
    if [[ -f "frontend/public/$f" ]]; then
      cp "frontend/public/$f" "$OUT/static/"
    fi
  done
fi

echo "==> Packaging migrations"
mkdir -p "$OUT/migrations"
cp -R internal/hub/repository/migrations/*.sql "$OUT/migrations/" 2>/dev/null || true

echo "==> Writing release metadata"
cat > "$OUT/RELEASE.txt" <<EOF
version=${VERSION}
commit=${COMMIT}
build_time=${BUILD_TIME}
artifact=hub-server (linux/amd64)
static=static/
migrations=migrations/
EOF

# tarball
TAR="dist/release/agent-hub-${VERSION}-linux-amd64.tar.gz"
mkdir -p dist/release
tar -C "$OUT" -czf "$TAR" .
echo "==> Wrote $TAR"
ls -la "$OUT" "$TAR"
echo "OK ${VERSION}"
