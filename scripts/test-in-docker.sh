#!/usr/bin/env bash
# ==========================================================
#   Agent Hub Docker Integration Test Sandbox (Bash)
# ==========================================================

set -euo pipefail

COMPOSE_FILE="docker-compose.test.yml"
USE_CONTAINER_SERVER=0
SKIP_CLEANUP=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --docker-hub|--container)
      USE_CONTAINER_SERVER=1
      shift
      ;;
    --skip-cleanup)
      SKIP_CLEANUP=1
      shift
      ;;
    --compose-file)
      COMPOSE_FILE="$2"
      shift 2
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

echo "=========================================================="
echo "  Agent Hub Docker Integration Test Sandbox (Bash)        "
echo "=========================================================="
echo "Repo Root: $REPO_ROOT"
echo "Compose:   $COMPOSE_FILE"

# 检测 docker compose 命令
if docker compose version >/dev/null 2>&1; then
  COMPOSE_CMD="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE_CMD="docker-compose"
else
  echo "Error: Neither 'docker compose' nor 'docker-compose' found."
  exit 1
fi

HUB_PID=""
TEMP_BIN_DIR="$REPO_ROOT/.tmp-bin"
TEMP_BIN_PATH=""
TEST_EXIT_CODE=1

cleanup() {
  local exit_code=$?
  if [ "$SKIP_CLEANUP" -eq 1 ]; then
    echo -e "\n[CLEANUP] Skipped (--skip-cleanup specified)."
    exit $exit_code
  fi

  echo -e "\n[CLEANUP] Tearing down test containers and volumes..."
  if [ -n "$HUB_PID" ] && kill -0 "$HUB_PID" 2>/dev/null; then
    echo "  Stopping local hub-server process ($HUB_PID)..."
    kill "$HUB_PID" 2>/dev/null || true
    wait "$HUB_PID" 2>/dev/null || true
  fi

  if [ -d "$TEMP_BIN_DIR" ]; then
    rm -rf "$TEMP_BIN_DIR" 2>/dev/null || true
  fi

  $COMPOSE_CMD -f "$COMPOSE_FILE" --profile full down -v >/dev/null 2>&1 || true
  echo "  Sandbox cleaned up successfully."
  exit $TEST_EXIT_CODE
}

trap cleanup EXIT INT TERM

# 1. 启动测试数据库与缓存容器
echo -e "\n[1/5] Starting dependencies (Postgres & Redis)..."
$COMPOSE_CMD -f "$COMPOSE_FILE" up -d test-postgres test-redis

# 2. 等待 Postgres 就绪
echo -e "\n[2/5] Waiting for PostgreSQL to be healthy..."
PG_READY=0
for i in $(seq 1 30); do
  if docker exec hub-test-postgres pg_isready -U hub_test -d hub_test >/dev/null 2>&1; then
    PG_READY=1
    echo "  PostgreSQL is healthy after ${i} second(s)."
    break
  fi
  sleep 1
done

if [ "$PG_READY" -ne 1 ]; then
  echo "Error: PostgreSQL failed to become healthy within 30 seconds."
  exit 1
fi

# 3. 启动 Hub Server (容器或本地二进制)
echo -e "\n[3/5] Starting Hub Server..."
HAS_LOCAL_GO=0
if command -v go >/dev/null 2>&1; then
  HAS_LOCAL_GO=1
fi

if [ "$USE_CONTAINER_SERVER" -eq 1 ] || [ "$HAS_LOCAL_GO" -eq 0 ]; then
  echo "  Mode: Docker container (golang:1.25.7)"
  $COMPOSE_CMD -f "$COMPOSE_FILE" --profile full up -d test-hub
else
  echo "  Mode: Local Go process (connecting to container DB/Redis)"
  mkdir -p "$TEMP_BIN_DIR"
  TEMP_BIN_PATH="$TEMP_BIN_DIR/hub-test-runner-$$"
  echo "  Compiling temporary binary $TEMP_BIN_PATH..."
  go build -o "$TEMP_BIN_PATH" ./cmd/hub

  export HUB_DATABASE_URL="postgres://hub_test:hub_password@localhost:15432/hub_test?sslmode=disable&search_path=hub,public"
  export HUB_REDIS_URL="redis://localhost:16379/0"
  export HUB_JWT_SECRET="test-secret-key-must-be-long-enough-32bytes"
  export PORT="9000"
  export HUB_PORT="9000"
  export HUB_HOST="127.0.0.1"

  "$TEMP_BIN_PATH" &
  HUB_PID=$!
fi

# 4. 等待 Hub Server 迁移与健康就绪
echo "  Waiting for migrations & http://127.0.0.1:9000/health..."
HUB_READY=0
for i in $(seq 1 45); do
  if curl -sf http://127.0.0.1:9000/health >/dev/null 2>&1; then
    if docker exec hub-test-postgres psql -U hub_test -d hub_test -c "SELECT 1 FROM hub.hub_businesses LIMIT 1;" >/dev/null 2>&1; then
      HUB_READY=1
      echo "  Hub Server and migrations are ready after ${i} second(s)."
      break
    fi
  fi
  sleep 1
done

if [ "$HUB_READY" -ne 1 ]; then
  echo "Error: Hub Server or migrations failed to become ready within 45 seconds."
  exit 1
fi

# 5. 预置测试租户与测试 API Key
echo -e "\n[4/5] Seeding test business and API key..."
SEED_SQL="
INSERT INTO hub.hub_businesses (code, name, description, status) 
VALUES ('test', 'Test Workspace', 'Docker Test Sandbox', 'active') 
ON CONFLICT (code) DO NOTHING;

INSERT INTO hub.hub_api_keys (business_id, key_hash, label) 
SELECT id, '01a5985ff3ed9a5779ebb70a70f0a631686e7f7a0d859443e3da71d1a0864327', 'test-key' 
FROM hub.hub_businesses WHERE code = 'test' 
ON CONFLICT DO NOTHING;
"

echo "$SEED_SQL" | docker exec -i hub-test-postgres psql -U hub_test -d hub_test -q
echo "  Seed completed: Business 'test' and test API Key ready."

# 6. 执行集成测试套件
echo -e "\n[5/5] Executing API Smoke Test Suite (scripts/test-api.js)..."
export HUB_PROTOCOL="http"
export HUB_HOST="127.0.0.1"
export HUB_PORT="9000"
export HUB_BUSINESS_CODE="test"
export HUB_API_KEY="test-api-key-for-docker-sandbox-32ch"

set +e
TEST_OUTPUT=$(node scripts/test-api.js 2>&1)
RAW_EXIT_CODE=$?
set -e

echo "$TEST_OUTPUT"

if echo "$TEST_OUTPUT" | grep -q "[0-9]\+ passed, 0 failed"; then
  echo -e "\n>>> All Integration Tests PASSED! <<<"
  TEST_EXIT_CODE=0
else
  echo -e "\n>>> Integration Tests FAILED! <<<"
  TEST_EXIT_CODE=1
fi
