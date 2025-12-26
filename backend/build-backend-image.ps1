# backend\build-backend-image.ps1
param(
  # Example explicit image: asia-northeast1-docker.pkg.dev/<PROJECT>/<REPO>/<SERVICE>:<TAG>
  [string]$Image,

  # Region / Cloud Run service name（イメージ命名にも使う）
  [string]$Region      = "asia-northeast1",
  [string]$ServiceName = "narratives-backend",

  # Artifact Registry repository name (Docker)
  [string]$RepoName    = "narratives-backend",

  # If -Image is auto-generated and Docker is available, use local docker build/push.
  # If Docker is unavailable, fallback to Cloud Build automatically.
  [bool]$PreferDockerWhenAvailable = $true
)

$ErrorActionPreference = "Stop"

function Write-Step($msg) { Write-Host "== $msg ==" -ForegroundColor Cyan }
function Write-Ok($msg)   { Write-Host "OK: $msg" -ForegroundColor Green }
function Write-Warn($msg) { Write-Host "!! $msg" -ForegroundColor Yellow }

function Test-DockerAvailable {
  try {
    docker info | Out-Null
    return $true
  } catch {
    return $false
  }
}

# ------------------------------------------------------------
# 0) gcloud のプロジェクト確認
# ------------------------------------------------------------
$ProjectId = (gcloud config get-value project 2>$null).Trim()
if (-not $ProjectId) {
  throw "gcloud config project is not set. Example: gcloud config set project <YOUR_PROJECT_ID>"
}
Write-Step "Using project: $ProjectId"

# スクリプト / ソースディレクトリ（backend/ 配下を想定）
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$SourceDir = $ScriptDir

# ------------------------------------------------------------
# 1) Go build チェック
# ------------------------------------------------------------
$MainGo = Join-Path $SourceDir "cmd\api\main.go"
if (-not (Test-Path $MainGo)) {
  throw "Go main file not found: $MainGo"
}

Write-Step "go build check (cmd/api)"
Push-Location $SourceDir
try {
  go version | Out-Null
  go build ./cmd/api
} finally {
  Pop-Location
}
Write-Ok "go build succeeded"

# ------------------------------------------------------------
# 2) Artifact Registry リポジトリ確認
# ------------------------------------------------------------
Write-Step "Ensuring Artifact Registry repository: ${RepoName}"
$repoExists = $true
try {
  gcloud artifacts repositories describe $RepoName --location=$Region --project=$ProjectId | Out-Null
} catch {
  $repoExists = $false
}

if (-not $repoExists) {
  Write-Warn "Repository '${RepoName}' not found. Creating it..."
  gcloud artifacts repositories create $RepoName `
    --repository-format=docker `
    --location=$Region `
    --description="Backend images for ${ServiceName}" `
    --project=$ProjectId | Out-Null
  Write-Ok "Repository created: ${RepoName}"
} else {
  Write-Ok "Repository exists: ${RepoName}"
}

# ------------------------------------------------------------
# 3) イメージ名決定（未指定なら自動生成）
# ------------------------------------------------------------
if ([string]::IsNullOrWhiteSpace($Image)) {
  $RegistryHost = "${Region}-docker.pkg.dev"
  $Tag = Get-Date -Format "yyyyMMdd-HHmmss"
  $Image = "${RegistryHost}/${ProjectId}/${RepoName}/${ServiceName}:${Tag}"
  Write-Step "No image specified. Generated image: $Image"
} else {
  Write-Step "Using specified image: $Image"
}

# ------------------------------------------------------------
# 4) build & push（Docker 優先 / fallback Cloud Build）
# ------------------------------------------------------------
$useDocker = $false
if ($PreferDockerWhenAvailable) {
  $useDocker = Test-DockerAvailable
}

if ($useDocker) {
  Write-Step "Docker detected. Using local docker build & push"

  $RegistryHost = "${Region}-docker.pkg.dev"
  Write-Step "Configuring Docker auth for ${RegistryHost}"
  gcloud auth configure-docker "$RegistryHost" -q | Out-Null
  Write-Ok "Docker auth configured"

  Push-Location $SourceDir
  try {
    docker build -t "$Image" .
    if ($LASTEXITCODE -ne 0) { throw "docker build failed with exit code $LASTEXITCODE" }

    docker push "$Image"
    if ($LASTEXITCODE -ne 0) { throw "docker push failed with exit code $LASTEXITCODE" }
  } finally {
    Pop-Location
  }
  Write-Ok "Image build & push completed (Docker)"
} else {
  Write-Step "Docker not available (or disabled). Using Cloud Build"
  gcloud builds submit --tag "$Image" --project "$ProjectId"
  if ($LASTEXITCODE -ne 0) {
    throw "Cloud Build failed. exit code: $LASTEXITCODE"
  }
  Write-Ok "Image build & push completed (Cloud Build)"
}

# ✅ 呼び出し元で受け取れるように最後に出力
Write-Output $Image
