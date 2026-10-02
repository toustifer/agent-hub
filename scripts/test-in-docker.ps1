[CmdletBinding()]
param(
    [string]$ComposeFile = "docker-compose.test.yml",
    [switch]$UseContainerServer,
    [switch]$SkipCleanup
)

$ErrorActionPreference = "Stop"

# 1. 确保定位到仓库根目录
$repoRoot = (Resolve-Path "$PSScriptRoot\..").Path
Set-Location $repoRoot

Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "  Agent Hub Docker Integration Test Sandbox (PowerShell)  " -ForegroundColor Cyan
Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "Repo Root: $repoRoot"
Write-Host "Compose:   $ComposeFile"

# 2. 检查 Docker 与 Compose 命令
$composeCmd = $null
if (Get-Command "docker" -ErrorAction SilentlyContinue) {
    $composeCheck = docker compose version 2>&1
    if ($LASTEXITCODE -eq 0) {
        $composeCmd = "docker compose"
    } elseif (Get-Command "docker-compose" -ErrorAction SilentlyContinue) {
        $composeCmd = "docker-compose"
    }
}

if (-not $composeCmd) {
    Write-Error "Docker Compose is required but neither 'docker compose' nor 'docker-compose' was found."
    exit 1
}

function Stop-PortProcess([int]$port) {
    try {
        $conns = Get-NetTCPConnection -LocalPort $port -ErrorAction SilentlyContinue
        foreach ($conn in $conns) {
            if ($conn.OwningProcess -and $conn.OwningProcess -gt 0) {
                Stop-Process -Id $conn.OwningProcess -Force -ErrorAction SilentlyContinue
            }
        }
    } catch {}
}

# 确保端口 9000 干净
Stop-PortProcess 9000

$hubProcess = $null
$tempBinaryPath = $null
$testExitCode = 1

try {
    # 3. 启动基础设施容器 (Postgres 16, Redis 7)
    Write-Host "`n[1/5] Starting dependencies (Postgres & Redis)..." -ForegroundColor Yellow
    Invoke-Expression "$composeCmd -f $ComposeFile up -d test-postgres test-redis"
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to start test containers."
    }

    # 4. 等待数据库健康就绪
    Write-Host "`n[2/5] Waiting for PostgreSQL to be healthy..." -ForegroundColor Yellow
    $pgReady = $false
    for ($i = 1; $i -le 30; $i++) {
        $check = docker exec hub-test-postgres pg_isready -U hub_test -d hub_test 2>&1
        if ($LASTEXITCODE -eq 0) {
            $pgReady = $true
            Write-Host "  PostgreSQL is healthy after $i second(s)." -ForegroundColor Green
            break
        }
        Start-Sleep -Seconds 1
    }
    if (-not $pgReady) {
        throw "PostgreSQL failed to become healthy within 30 seconds."
    }

    # 5. 启动 Hub Server (容器内或本地二进制)
    Write-Host "`n[3/5] Starting Hub Server..." -ForegroundColor Yellow
    $hasLocalGo = [bool](Get-Command "go" -ErrorAction SilentlyContinue)
    $runInContainer = $UseContainerServer -or (-not $hasLocalGo)

    if ($runInContainer) {
        Write-Host "  Mode: Docker container (golang:1.25.7)" -ForegroundColor Cyan
        Invoke-Expression "$composeCmd -f $ComposeFile --profile full up -d test-hub"
        if ($LASTEXITCODE -ne 0) {
            throw "Failed to start test-hub container."
        }
    } else {
        Write-Host "  Mode: Local Go process (connecting to container DB/Redis)" -ForegroundColor Cyan
        $binDir = Join-Path $repoRoot ".tmp-bin"
        if (-not (Test-Path $binDir)) { New-Item -ItemType Directory -Path $binDir | Out-Null }
        $tempBinaryPath = Join-Path $binDir "hub-test-runner-$PID.exe"
        
        Write-Host "  Building temporary binary $tempBinaryPath..."
        go build -o $tempBinaryPath ./cmd/hub
        if ($LASTEXITCODE -ne 0) {
            throw "Failed to compile hub server binary."
        }

        $env:HUB_DATABASE_URL = "postgres://hub_test:hub_password@localhost:15432/hub_test?sslmode=disable&search_path=hub,public"
        $env:HUB_REDIS_URL = "redis://localhost:16379/0"
        $env:HUB_JWT_SECRET = "test-secret-key-must-be-long-enough-32bytes"
        $env:PORT = "9000"
        $env:HUB_PORT = "9000"
        $env:HUB_HOST = "127.0.0.1"

        $hubProcess = Start-Process -FilePath $tempBinaryPath -PassThru -NoNewWindow
    }

    # 6. 等待 Hub Server 迁移与健康就绪
    Write-Host "  Waiting for migrations & http://127.0.0.1:9000/health..." -ForegroundColor Yellow
    $hubReady = $false
    for ($i = 1; $i -le 45; $i++) {
        # 检查健康检查与迁移完成标志
        try {
            $resp = Invoke-RestMethod -Uri "http://127.0.0.1:9000/health" -Method Get -TimeoutSec 2 -ErrorAction SilentlyContinue
            if ($resp -and ($resp.status -eq "healthy" -or $resp.code -eq 200)) {
                $tableCheck = docker exec hub-test-postgres psql -U hub_test -d hub_test -c "SELECT 1 FROM hub.hub_businesses LIMIT 1;" 2>&1
                if ($LASTEXITCODE -eq 0) {
                    $hubReady = $true
                    Write-Host "  Hub Server and migrations are ready after $i second(s)." -ForegroundColor Green
                    break
                }
            }
        } catch {}
        Start-Sleep -Seconds 1
    }
    if (-not $hubReady) {
        throw "Hub Server or migrations failed to become ready within 45 seconds."
    }

    # 7. 预置测试租户与测试 API Key
    Write-Host "`n[4/5] Seeding test business and API key..." -ForegroundColor Yellow
    $seedSql = @"
INSERT INTO hub.hub_businesses (code, name, description, status) 
VALUES ('test', 'Test Workspace', 'Docker Test Sandbox', 'active') 
ON CONFLICT (code) DO NOTHING;

INSERT INTO hub.hub_api_keys (business_id, key_hash, label) 
SELECT id, '01a5985ff3ed9a5779ebb70a70f0a631686e7f7a0d859443e3da71d1a0864327', 'test-key' 
FROM hub.hub_businesses WHERE code = 'test' 
ON CONFLICT DO NOTHING;
"@
    $seedSql | docker exec -i hub-test-postgres psql -U hub_test -d hub_test -q
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to seed test tenant data."
    }
    Write-Host "  Seed completed: Business 'test' and test API Key ready." -ForegroundColor Green

    # 8. 执行集成测试套件
    Write-Host "`n[5/5] Executing API Smoke Test Suite (scripts/test-api.js)..." -ForegroundColor Yellow
    $env:HUB_PROTOCOL = "http"
    $env:HUB_HOST = "127.0.0.1"
    $env:HUB_PORT = "9000"
    $env:HUB_BUSINESS_CODE = "test"
    $env:HUB_API_KEY = "test-api-key-for-docker-sandbox-32ch"

    $testOutput = node scripts/test-api.js 2>&1
    $testExitCode = $LASTEXITCODE

    Write-Host ($testOutput -join "`n")

    if ($testOutput -match "(\d+) passed, 0 failed") {
        Write-Host "`n>>> All Integration Tests PASSED! <<<" -ForegroundColor Green
        $testExitCode = 0
    } else {
        Write-Host "`n>>> Integration Tests FAILED! <<<" -ForegroundColor Red
        $testExitCode = 1
    }

} catch {
    Write-Error "Sandbox execution failed: $_"
    $testExitCode = 1
} finally {
    # 9. 资源清理
    if ($SkipCleanup) {
        Write-Host "`n[CLEANUP] Skipped (--SkipCleanup specified)." -ForegroundColor DarkYellow
    } else {
        Write-Host "`n[CLEANUP] Tearing down test containers and volumes..." -ForegroundColor Yellow
        if ($hubProcess -and -not $hubProcess.HasExited) {
            Write-Host "  Stopping local hub-server process ($($hubProcess.Id))..."
            Stop-Process -Id $hubProcess.Id -Force -ErrorAction SilentlyContinue
        }
        Stop-PortProcess 9000

        if ($tempBinaryPath -and (Test-Path $tempBinaryPath)) {
            Remove-Item -Force $tempBinaryPath -ErrorAction SilentlyContinue
        }
        $binDir = Join-Path $repoRoot ".tmp-bin"
        if (Test-Path $binDir) {
            Remove-Item -Recurse -Force $binDir -ErrorAction SilentlyContinue
        }

        Invoke-Expression "$composeCmd -f $ComposeFile --profile full down -v" | Out-Null
        Write-Host "  Sandbox cleaned up successfully." -ForegroundColor Green
    }

    exit $testExitCode
}
