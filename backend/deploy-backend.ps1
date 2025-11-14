# backend/deploy-backend.ps1
param(
  [Parameter(Mandatory = $true)]
  [string]$Image,                   # 例: asia-northeast1-docker.pkg.dev/.../narratives-backend:20251113173405
  [string]$Region      = "asia-northeast1",
  [string]$ServiceName = "narratives-backend"
)

$ErrorActionPreference = "Stop"

function Write-Step($msg) { Write-Host "== $msg ==" -ForegroundColor Cyan }
function Write-Ok($msg)   { Write-Host "OK: $msg" -ForegroundColor Green }

# gcloud のアクティブ プロジェクトを取得
$ProjectId = (gcloud config get-value project 2>$null).Trim()
if (-not $ProjectId) {
  throw "gcloud config project が設定されていません。`n 例: gcloud config set project narratives-development-26c2d"
}

# Cloud Run 実行用サービスアカウント
$RunServiceAccount = "narratives-backend-sa@$ProjectId.iam.gserviceaccount.com"

# backend/cmd/api/main.go の存在チェック
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$SourceDir = $ScriptDir
$MainGo    = Join-Path $SourceDir "cmd\api\main.go"
if (-not (Test-Path $MainGo)) {
  throw "Go メインファイルが見つかりません: $MainGo"
}

# 1. go build チェック（ローカルでコンパイルできるか確認）
Write-Step "go build チェック (cmd/api)"

Push-Location $SourceDir
try {
  go version | Out-Null
  go build ./cmd/api
} finally {
  Pop-Location
}

Write-Ok "go build 成功 (コンパイル OK)"

# 2. Cloud Run へデプロイ
Write-Step "Cloud Run へデプロイ"

$deployArgs = @(
  "run","deploy", $ServiceName,
  "--image",          $Image,
  "--region",         $Region,
  "--platform",       "managed",
  "--allow-unauthenticated",
  "--service-account", $RunServiceAccount,
  "--min-instances",  "0",
  "--max-instances",  "5",
  "--memory",         "512Mi",
  "--cpu",            "1",
  "--concurrency",    "80",
  "--timeout",        "60s",
  "--project",        $ProjectId
)

& gcloud @deployArgs
if ($LASTEXITCODE -ne 0) {
  throw "gcloud run deploy が失敗しました。exit code: $LASTEXITCODE"
}

Write-Ok "Cloud Run へのデプロイ完了: サービス '$ServiceName'"
