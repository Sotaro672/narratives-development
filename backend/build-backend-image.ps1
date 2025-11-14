# backend/build-backend-image.ps1
param(
  [string]$Region = "asia-northeast1",
  [string]$Repo   = "backend"
)

$ErrorActionPreference = "Stop"

function Write-Step($msg) { Write-Host "== $msg ==" -ForegroundColor Cyan }
function Write-Ok($msg)   { Write-Host "OK: $msg" -ForegroundColor Green }

# gcloud のアクティブ プロジェクトを取得
$ProjectId = (gcloud config get-value project 2>$null).Trim()
if (-not $ProjectId) {
  throw "gcloud config project が設定されていません。`n 例: gcloud config set project narratives-development-26c2d"
}

# backend ディレクトリと Dockerfile 確認
$ScriptDir  = Split-Path -Parent $MyInvocation.MyCommand.Path
$SourceDir  = $ScriptDir               # backend フォルダ
$Dockerfile = Join-Path $SourceDir "Dockerfile"
if (-not (Test-Path $Dockerfile)) {
  throw "Dockerfile が見つかりません: $Dockerfile"
}

# タグ付きイメージ名を生成
$Tag   = Get-Date -Format "yyyyMMddHHmmss"
$Image = "$Region-docker.pkg.dev/$ProjectId/$Repo/narratives-backend:$Tag"

Write-Step "API 有効化 (Artifact Registry / Cloud Build / Cloud Run)"
gcloud services enable `
  artifactregistry.googleapis.com `
  cloudbuild.googleapis.com `
  run.googleapis.com `
  --project $ProjectId

Write-Step "Artifact Registry リポジトリ '$Repo' の存在確認 ($Region)"
$repoExists = $false
try {
  gcloud artifacts repositories describe $Repo `
    --location $Region `
    --project $ProjectId | Out-Null
  $repoExists = $true
} catch {
  $repoExists = $false
}

if (-not $repoExists) {
  Write-Step "Artifact Registry リポジトリ '$Repo' を作成"
  gcloud artifacts repositories create $Repo `
    --repository-format=docker `
    --location $Region `
    --description "Backend images" `
    --project $ProjectId
}

Write-Step "Cloud Build でイメージをビルド: $Image"

Push-Location $SourceDir
try {
  gcloud builds submit `
    --tag $Image `
    --project $ProjectId
} finally {
  Pop-Location
}

Write-Ok "イメージビルド完了: $Image"
Write-Host ""
Write-Host "このイメージを Cloud Run にデプロイする場合:" -ForegroundColor Yellow
Write-Host "  .\backend\deploy-backend.ps1 -Image `"$Image`"" -ForegroundColor Yellow
