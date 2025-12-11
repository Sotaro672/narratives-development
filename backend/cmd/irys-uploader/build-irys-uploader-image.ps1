param(
  [string]$Region = "asia-northeast1",          # Artifact Registry region
  [string]$Repo   = "narratives-irys-uploader"  # Artifact Registry repository name
)

$ErrorActionPreference = "Stop"

function Write-Step($msg) { Write-Host "== $msg ==" -ForegroundColor Cyan }
function Write-Ok($msg)   { Write-Host "OK: $msg" -ForegroundColor Green }
function Write-Warn($msg) { Write-Host "!! $msg !!" -ForegroundColor Yellow }

# ------------------------------------------------------------
# 0) gcloud のプロジェクト確認
# ------------------------------------------------------------
$ProjectId = (gcloud config get-value project 2>$null).Trim()
if (-not $ProjectId) {
  throw "gcloud config project is not set. Example: gcloud config set project narratives-development-26c2d"
}
Write-Step "Using project: $ProjectId"

# スクリプトディレクトリ: ...\backend\cmd\irys-uploader
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
# cmd ディレクトリ: ...\backend\cmd
$CmdDir    = Split-Path -Parent $ScriptDir
# プロジェクトルート: ...\backend
$RootDir   = Split-Path -Parent $CmdDir

# Docker build のコンテキストはプロジェクトルート
$SourceDir  = $RootDir
$Dockerfile = Join-Path $RootDir "cmd\irys-uploader\Dockerfile"

if (-not (Test-Path $Dockerfile)) {
  throw "Dockerfile not found: $Dockerfile"
}

# ------------------------------------------------------------
# 1) 必要な API を有効化
# ------------------------------------------------------------
Write-Step "Enable required APIs (Artifact Registry / Cloud Build / Cloud Run)"
gcloud services enable `
  artifactregistry.googleapis.com `
  cloudbuild.googleapis.com `
  run.googleapis.com `
  --project $ProjectId
Write-Ok "Required APIs enabled (or already enabled)"

# ------------------------------------------------------------
# 2) Artifact Registry リポジトリ確認
# ------------------------------------------------------------
Write-Step "Check Artifact Registry repository '$Repo' in region '$Region'"
$repoExists = $true
try {
  gcloud artifacts repositories describe $Repo `
    --location $Region `
    --project $ProjectId | Out-Null
} catch {
  $repoExists = $false
}

if (-not $repoExists) {
  Write-Step "Create Artifact Registry repository '$Repo'"
  gcloud artifacts repositories create $Repo `
    --repository-format=docker `
    --location $Region `
    --description "Irys uploader images" `
    --project $ProjectId
  Write-Ok "Repository created: $Repo"
} else {
  Write-Ok "Repository exists: $Repo"
}

# ------------------------------------------------------------
# 3) イメージ名生成
# ------------------------------------------------------------
$Tag   = Get-Date -Format "yyyyMMddHHmmss"
$Image = "$Region-docker.pkg.dev/$ProjectId/$Repo/narratives-irys-uploader:$Tag"
Write-Step "Image to build: $Image"

# ------------------------------------------------------------
# 4) Docker で build & push
# ------------------------------------------------------------
function Test-DockerAvailable {
  try {
    docker info | Out-Null
    return $true
  } catch {
    return $false
  }
}

if (-not (Test-DockerAvailable)) {
  throw "Docker is not available. Please install Docker Desktop or ensure 'docker info' works."
}

$RegistryHost = "$Region-docker.pkg.dev"
Write-Step "Configuring Docker auth for $RegistryHost"
gcloud auth configure-docker "$RegistryHost" -q | Out-Null
Write-Ok "Docker auth configured"

Write-Step "Building Docker image (context=$SourceDir, dockerfile=$Dockerfile)"
Push-Location $SourceDir
try {
  docker build -f "cmd/irys-uploader/Dockerfile" -t "$Image" .
  if ($LASTEXITCODE -ne 0) {
    throw "docker build failed with exit code $LASTEXITCODE"
  }

  Write-Step "Pushing Docker image to Artifact Registry"
  docker push "$Image"
  if ($LASTEXITCODE -ne 0) {
    throw "docker push failed with exit code $LASTEXITCODE"
  }
} finally {
  Pop-Location
}

Write-Ok "Image build & push completed: $Image"
Write-Host ""
Write-Host "To deploy this image to Cloud Run:" -ForegroundColor Yellow
Write-Host "  ..\..\deploy-irys-uploader.ps1 -Image `"$Image`"" -ForegroundColor Yellow
