# backend\cmd\irys-uploader\deploy-irys-uploader.ps1
param(
  # 例: asia-northeast1-docker.pkg.dev/<PROJECT>/narratives-irys-uploader/narratives-irys-uploader:20251211010101
  # 省略した場合は Artifact Registry から「最新の」イメージを自動解決する
  [string]$Image,

  [string]$Region      = "asia-northeast1",
  [string]$ServiceName = "narratives-irys-uploader"
)

$ErrorActionPreference = "Stop"

function Write-Step($msg) { Write-Host "== $msg ==" -ForegroundColor Cyan }
function Write-Ok($msg)   { Write-Host "OK: $msg" -ForegroundColor Green }
function Write-Warn($msg) { Write-Host "!! $msg ==" -ForegroundColor Yellow }

# ------------------------------------------------------------
# 0) gcloud のプロジェクト確認
# ------------------------------------------------------------
$ProjectId = (gcloud config get-value project 2>$null).Trim()
if (-not $ProjectId) {
  throw "gcloud config project is not set. Example: gcloud config set project narratives-development-26c2d"
}
Write-Step "Using project: $ProjectId"

# Cloud Run service account（backend と同じ SA を再利用）
$RunServiceAccount = "narratives-backend-sa@$ProjectId.iam.gserviceaccount.com"

# スクリプト / ソースディレクトリ
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$SourceDir = $ScriptDir

# ------------------------------------------------------------
# 0.5) Image が指定されていない場合は Artifact Registry から最新を自動解決
# ------------------------------------------------------------
#   - build-irys-uploader-image.ps1 が付けているタグ:
#       asia-northeast1-docker.pkg.dev/<PROJECT>/narratives-irys-uploader/narratives-irys-uploader:<TIMESTAMP>
#   - ここではその中で UPDATE_TIME が最新のものを 1 件拾う
# ------------------------------------------------------------
$RepoName  = "narratives-irys-uploader"
$ImageName = "narratives-irys-uploader"

if ([string]::IsNullOrWhiteSpace($Image)) {
  Write-Step "No -Image specified. Resolving latest image from Artifact Registry..."

  $RegistryHost  = "${Region}-docker.pkg.dev"
  $RepoPath      = "$RegistryHost/$ProjectId/$RepoName"
  $PackageFilter = "$RegistryHost/$ProjectId/$RepoName/$ImageName"

  # StackOverflow 推奨パターン:
  # gcloud artifacts docker images list ${REGION}-docker.pkg.dev/${PROJECT}/${REPO} \
  #   --filter="package=${REGION}-docker.pkg.dev/${PROJECT}/${REPO}/${IMAGE}" \
  #   --sort-by="~UPDATE_TIME" \
  #   --limit=1 \
  #   --format="value(format("{0}@{1}",package,version))"
  $Image = (& gcloud artifacts docker images list $RepoPath `
      --filter="package=$PackageFilter" `
      --sort-by="~UPDATE_TIME" `
      --limit=1 `
      --format='value(format("{0}@{1}",package,version))').Trim()

  if ([string]::IsNullOrWhiteSpace($Image)) {
    throw "No images found in $RepoPath for package $PackageFilter. Run .\build-irys-uploader-image.ps1 first."
  }

  Write-Ok "Resolved latest image: $Image"
} else {
  Write-Step "Using specified image: $Image"
}

# ------------------------------------------------------------
# 1) Cloud Run に渡す環境変数を組み立てる
# ------------------------------------------------------------
Write-Step "Collecting env vars for Cloud Run"

# 必ずセットしておくと便利なもの
$envPairs = @("GOOGLE_CLOUD_PROJECT=$ProjectId")

# backend/cmd/irys-uploader/.env を読む想定
$EnvFile = Join-Path $SourceDir ".env"
if (Test-Path $EnvFile) {
  Write-Ok "Found .env: $EnvFile"

  foreach ($line in Get-Content $EnvFile) {
    $trim = $line.Trim()
    if (-not $trim) { continue }
    if ($trim.StartsWith("#")) { continue }

    $idx = $trim.IndexOf("=")
    if ($idx -lt 1) { continue }

    $key   = $trim.Substring(0, $idx).Trim()
    $value = $trim.Substring($idx + 1).Trim()

    # Irys uploader 用に渡したいキーだけをピックアップ
    # （IRYS_* プレフィックスを全部渡す）
    if ($key -like "IRYS_*") {
      $envPairs += "$key=$value"
    }
  }
} else {
  Write-Warn ".env file not found at $EnvFile. Only GOOGLE_CLOUD_PROJECT will be set."
}

$envArg = [string]::Join(",", $envPairs)
Write-Step "Env vars to set: $envArg"

# ------------------------------------------------------------
# 2) Cloud Run へデプロイ
# ------------------------------------------------------------
Write-Step "Deploying Irys uploader to Cloud Run"

$deployArgs = @(
  "run","deploy", $ServiceName,
  "--image",          $Image,
  "--region",         $Region,
  "--platform",       "managed",
  "--allow-unauthenticated",
  "--service-account", $RunServiceAccount,
  "--set-env-vars",    $envArg,
  "--min-instances",  "0",
  "--max-instances",  "2",
  "--memory",         "256Mi",
  "--cpu",            "1",
  "--concurrency",    "10",
  "--timeout",        "60s",
  "--project",        $ProjectId
)

& gcloud @deployArgs
if ($LASTEXITCODE -ne 0) {
  throw "gcloud run deploy failed. exit code: $LASTEXITCODE"
}

Write-Ok "Cloud Run deployment finished: service '${ServiceName}'"
Write-Ok "Deployed with image: ${Image}"
