param(
  [string]$Region = "asia-northeast1",   # Artifact Registry region
  [string]$Repo   = "backend"            # Artifact Registry repository name
)

$ErrorActionPreference = "Stop"

function Write-Step($msg) { Write-Host "== $msg ==" -ForegroundColor Cyan }
function Write-Ok($msg)   { Write-Host "OK: $msg" -ForegroundColor Green }
function Write-Warn($msg) { Write-Host "!! $msg" -ForegroundColor Yellow }

# Resolve active gcloud project
$ProjectId = (gcloud config get-value project 2>$null).Trim()
if (-not $ProjectId) {
  throw "gcloud config project is not set. Example: gcloud config set project narratives-development-26c2d"
}

# Validate Dockerfile location (backend folder)
$ScriptDir  = Split-Path -Parent $MyInvocation.MyCommand.Path
$SourceDir  = $ScriptDir
$Dockerfile = Join-Path $SourceDir "Dockerfile"
if (-not (Test-Path $Dockerfile)) {
  throw "Dockerfile not found: $Dockerfile"
}

# Generate a tagged image name
$Tag   = Get-Date -Format "yyyyMMddHHmmss"
$Image = "$Region-docker.pkg.dev/$ProjectId/$Repo/narratives-backend:$Tag"

Write-Step "Enable required APIs (Artifact Registry / Cloud Build / Cloud Run)"
gcloud services enable `
  artifactregistry.googleapis.com `
  cloudbuild.googleapis.com `
  run.googleapis.com `
  --project $ProjectId

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
    --description "Backend images" `
    --project $ProjectId
  Write-Ok "Repository created: $Repo"
} else {
  Write-Ok "Repository exists: $Repo"
}

Write-Step "Build image with Cloud Build: $Image"
Push-Location $SourceDir
try {
  gcloud builds submit `
    --tag $Image `
    --project $ProjectId
} finally {
  Pop-Location
}

Write-Ok "Image build completed: $Image"
Write-Host ""
Write-Host "Built image (copy & paste if needed):" -ForegroundColor Yellow
Write-Host "  $Image" -ForegroundColor Cyan
Write-Host ""
Write-Host "To deploy this image to Cloud Run:" -ForegroundColor Yellow
Write-Host "  .\backend\deploy-backend.ps1 -Image `"$Image`"" -ForegroundColor Yellow
