#!/usr/bin/env bash
# Deploy a release tarball (or local dist/release/vX) to production host.
#
# Usage:
#   ./scripts/deploy-release.sh v0.2.0
#   ./scripts/deploy-release.sh v0.2.0 storyhost
#   HUB_DEPLOY_HOST=storyhost HUB_DEPLOY_PATH=/opt/agent-hub ./scripts/deploy-release.sh v0.2.0
#
# Expects:
#   - SSH host alias or user@host with access to production
#   - Remote path with hub-server, .env, static/
#   - systemd unit hub-server.service (preferred)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION="${1:-}"
HOST="${2:-${HUB_DEPLOY_HOST:-storyhost}}"
REMOTE="${HUB_DEPLOY_PATH:-/opt/agent-hub}"

if [[ -z "$VERSION" ]]; then
  echo "usage: $0 <version> [ssh-host]" >&2
  echo "  e.g. $0 v0.2.0 storyhost" >&2
  exit 1
fi
[[ "$VERSION" == v* ]] || VERSION="v$VERSION"

TAR="dist/release/agent-hub-${VERSION}-linux-amd64.tar.gz"
DIR="dist/release/${VERSION}"

if [[ ! -f "$TAR" && ! -d "$DIR" ]]; then
  echo "missing artifacts; run: ./scripts/build-release.sh ${VERSION}" >&2
  exit 1
fi

echo "==> Deploy ${VERSION} → ${HOST}:${REMOTE}"

# upload
ssh -o BatchMode=yes "$HOST" "mkdir -p ${REMOTE}/releases ${REMOTE}/static ${REMOTE}/migrations"
if [[ -f "$TAR" ]]; then
  scp -o BatchMode=yes "$TAR" "${HOST}:${REMOTE}/releases/"
  ssh -o BatchMode=yes "$HOST" "set -e
    cd ${REMOTE}
    rm -rf releases/${VERSION}
    mkdir -p releases/${VERSION}
    tar -xzf releases/agent-hub-${VERSION}-linux-amd64.tar.gz -C releases/${VERSION}
  "
else
  scp -o BatchMode=yes -r "$DIR" "${HOST}:${REMOTE}/releases/"
fi

# apply migrations (all sql; IF NOT EXISTS safe) + swap binary + static + restart
ssh -o BatchMode=yes "$HOST" "set -e
  set -a
  # shellcheck disable=SC1091
  . ${REMOTE}/.env
  set +a
  cd ${REMOTE}
  REL=releases/${VERSION}
  test -x \$REL/hub-server

  echo '==> migrations'
  if command -v psql >/dev/null && [[ -n \"\${HUB_DATABASE_URL:-}\" ]]; then
    for f in \$(ls \$REL/migrations/*.sql 2>/dev/null | sort); do
      echo \"  apply \$(basename \$f)\"
      psql \"\$HUB_DATABASE_URL\" -v ON_ERROR_STOP=0 -f \"\$f\" >/tmp/hub-migrate-\$(basename \$f).log 2>&1 || true
    done
  else
    echo '  skip migrations (no psql or HUB_DATABASE_URL)'
  fi

  echo '==> backup + install binary'
  if [[ -f hub-server ]]; then
    cp -a hub-server hub-server.bak.\$(date +%Y%m%d%H%M%S)
  fi
  install -m 755 \$REL/hub-server hub-server
  ln -sfn \$REL current

  echo '==> static'
  if [[ -d \$REL/static ]]; then
    # keep a backup of previous index
    if [[ -f static/index.html ]]; then
      cp -a static/index.html static/index.html.bak.\$(date +%Y%m%d%H%M%S) 2>/dev/null || true
    fi
    cp -a \$REL/static/. static/
  fi

  echo '==> restart'
  if systemctl is-enabled hub-server.service >/dev/null 2>&1 || systemctl status hub-server.service >/dev/null 2>&1; then
    systemctl restart hub-server.service
    sleep 2
    systemctl is-active hub-server.service
  else
    pkill -x hub-server || true
    sleep 1
    nohup ./hub-server >>/var/log/agent-hub.log 2>&1 &
    sleep 2
    pgrep -x hub-server
  fi

  echo '==> smoke'
  curl -fsS http://127.0.0.1:9000/health
  echo
  curl -fsS http://127.0.0.1:9000/version || true
  echo
  echo DEPLOY_OK ${VERSION}
"

echo "==> Public smoke (if DNS ok)"
curl -fsS "https://hub.stifer.xyz/health" || true
echo
echo "DONE ${VERSION} on ${HOST}"
