param(
  # Example explicit image:
  # asia-northeast1-docker.pkg.dev/<PROJECT>/<REPO>/<SERVICE>:<TAG>
  [string]$Image,

  # Region / Cloud Run service name
  [string]$Region      = "asia-northeast1",
  [string]$ServiceName = "narratives-backend",

  # Artifact Registry repository name (Docker)
  [string]$RepoName    = "narratives-backend"
)

$ErrorActionPreference = "Stop"

function Write-Step($msg) { Write-Host "== $msg ==" -ForegroundColor Cyan }
function Write-Ok($msg)   { Write-Host "OK: $msg" -ForegroundColor Green }
function Write-Warn($msg) { Write-Host "!! $msg ==" -ForegroundColor Yellow }

function Normalize-EnvValue([string]$v) {
  if ($null -eq $v) { return "" }

  $s = $v
  if ($null -eq $s) { return "" }

  # strip surrounding quotes
  if (($s.StartsWith('"') -and $s.EndsWith('"')) -or ($s.StartsWith("'") -and $s.EndsWith("'"))) {
    $s = $s.Substring(1, $s.Length - 2)
  }

  return $s
}

function Read-EnvFile([string]$path) {
  $map = @{}

  foreach ($line in Get-Content $path) {
    if ($null -eq $line) { continue }

    $trim = $line
    if ($trim -eq "") { continue }
    if ($trim.StartsWith("#")) { continue }

    $idx = $trim.IndexOf("=")
    if ($idx -lt 1) { continue }

    $key   = $trim.Substring(0, $idx).Trim()
    $value = $trim.Substring($idx + 1)

    $map[$key] = (Normalize-EnvValue $value)
  }

  return $map
}

function Exec-GCloudOrThrow {
  param(
    [Parameter(Mandatory=$true)][string[]]$Args,
    [string]$ErrorMessage = "gcloud command failed."
  )

  & $GCLOUD @Args
  if ($LASTEXITCODE -ne 0) {
    throw "$ErrorMessage (exit code: $LASTEXITCODE) Args: gcloud $($Args -join ' ')"
  }
}

function Invoke-CloudBuildOrThrow {
  param(
    [Parameter(Mandatory=$true)][string]$Image
  )

  Write-Step "Running Cloud Build"
  Write-Step "Cloud Build image: $Image"

  Push-Location $SourceDir
  try {
    & $GCLOUD builds submit `
      --tag "$Image" `
      --project "$ProjectId"

    if ($LASTEXITCODE -ne 0) {
      throw "Cloud Build failed. exit code: $LASTEXITCODE"
    }
  } finally {
    Pop-Location
  }

  Write-Ok "Image build & push completed by Cloud Build"
}

# ------------------------------------------------------------
# 0) gcloud のプロジェクト/アカウント確認
# ------------------------------------------------------------
Write-Step "Starting deploy-backend.ps1"

$env:CLOUDSDK_CORE_DISABLE_PROMPTS = "1"
$env:CLOUDSDK_COMPONENT_MANAGER_DISABLE_UPDATE_CHECK = "1"

$GCLOUD = (Get-Command gcloud.cmd -ErrorAction Stop).Source
Write-Step "Using gcloud.cmd: $GCLOUD"

Write-Step "Confirming gcloud config (project/account)"
$ConfiguredProject = (& $GCLOUD config get-value project)
$ConfiguredAccount = (& $GCLOUD config get-value account)

if (-not $ConfiguredProject) {
  throw "gcloud config project is not set. Example: gcloud config set project <YOUR_PROJECT_ID>"
}

if (-not $ConfiguredAccount) {
  throw "gcloud active account is not set. Example: gcloud auth login"
}

Write-Ok "gcloud project: $ConfiguredProject"
Write-Ok "gcloud account: $ConfiguredAccount"

Write-Step "Resolving GCP project id"
$ProjectId = (& $GCLOUD config get-value project)
if (-not $ProjectId) {
  throw "gcloud config project is not set. Example: gcloud config set project <YOUR_PROJECT_ID>"
}

Write-Step "Using project: $ProjectId"

$RunServiceAccount = "narratives-backend-sa@$ProjectId.iam.gserviceaccount.com"

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

  if ($LASTEXITCODE -ne 0) {
    throw "go build ./cmd/api failed. exit code: $LASTEXITCODE"
  }
} finally {
  Pop-Location
}

Write-Ok "go build succeeded"

# ------------------------------------------------------------
# 2) Artifact Registry リポジトリ確認
# ------------------------------------------------------------
Write-Step "Ensuring Artifact Registry repository: ${RepoName}"

& $GCLOUD artifacts repositories describe $RepoName `
  --location=$Region `
  --project=$ProjectId | Out-Null

$repoExists = ($LASTEXITCODE -eq 0)

if (-not $repoExists) {
  Write-Warn "Repository '${RepoName}' not found OR no permission. Trying to create it..."

  & $GCLOUD artifacts repositories create $RepoName `
    --repository-format=docker `
    --location=$Region `
    --description="Backend images for ${ServiceName}" `
    --project=$ProjectId | Out-Null

  if ($LASTEXITCODE -ne 0) {
    throw "Failed to describe/create Artifact Registry repository '${RepoName}'. Check: (1) gcloud project/account, (2) IAM roles artifactregistry.*. exit code: $LASTEXITCODE"
  }

  Write-Ok "Repository created: ${RepoName}"
} else {
  Write-Ok "Repository exists: ${RepoName}"
}

# ------------------------------------------------------------
# 3) イメージ名決定
# ------------------------------------------------------------
$AutoGenerated = $false

if ([string]::IsNullOrWhiteSpace($Image)) {
  $RegistryHost = "${Region}-docker.pkg.dev"
  $Tag = Get-Date -Format "yyyyMMdd-HHmmss"
  $Image = "${RegistryHost}/${ProjectId}/${RepoName}/${ServiceName}:${Tag}"
  $AutoGenerated = $true

  Write-Step "No image specified. Generated image: $Image"
} else {
  Write-Step "Using specified image: $Image"
}

# ------------------------------------------------------------
# 4) Cloud Build でビルド & Artifact Registry へ push
# ------------------------------------------------------------
Invoke-CloudBuildOrThrow -Image $Image

# ------------------------------------------------------------
# 5) Cloud Run に渡す環境変数を組み立てる
#    - GCS bucket 系は廃止
#    - STRIPE_SECRET_KEY は Secret Manager の stripe-secret-key を使うため env には載せない
# ------------------------------------------------------------
Write-Step "Collecting env vars for Cloud Run"

$AllowedKeys = @(
  # Project / Firestore
  "GCP_PROJECT_ID",
  "FIREBASE_PROJECT_ID",
  "FIRESTORE_PROJECT_ID",

  # Resend
  "RESEND_API_KEY",
  "RESEND_FROM",
  "RESEND_CONTACT_ADMIN_TO",
  "CONSOLE_BASE_URL",

  # Solana
  "SOLANA_RPC_URL",
  "SOLANA_AIRDROP_RPC_URL",
  "SOLANA_AUTO_AIRDROP_ENABLED",
  "SOLANA_MIN_FEE_PAYER_BALANCE_SOL",
  "SOLANA_AIRDROP_AMOUNT_SOL",
  "SOLANA_MINT_KEY_SECRET",
  "SOLANA_SELLER_FEE_BPS",

  # Arweave / Irys
  "ARWEAVE_BASE_URL",

  # checkout self-callback base URL
  "SELF_BASE_URL",

  # Cloud Tasks / mint worker
  "CLOUD_TASKS_PROJECT_ID",
  "CLOUD_TASKS_LOCATION",
  "CLOUD_TASKS_QUEUE_ID",
  "INTERNAL_BASE_URL",
  "CLOUD_TASKS_SERVICE_ACCOUNT",
  "CLOUD_TASKS_AUDIENCE",
  "MINT_TASK_DISPATCH_DELAY_SECONDS",

  # Stripe webhook only
  # STRIPE_SECRET_KEY は廃止。Secret Manager の stripe-secret-key を使用する。
  "STRIPE_WEBHOOK_SECRET"
)

$envMap = @{}
$envMap["GOOGLE_CLOUD_PROJECT"] = $ProjectId

$EnvFile = Join-Path $SourceDir ".env"

if (Test-Path $EnvFile) {
  Write-Ok "Found .env: $EnvFile"
  $fileMap = Read-EnvFile $EnvFile

  foreach ($k in $AllowedKeys) {
    if ($fileMap.ContainsKey($k)) {
      $envMap[$k] = $fileMap[$k]
    }
  }
} else {
  Write-Warn ".env file not found at $EnvFile. Will only set GOOGLE_CLOUD_PROJECT and project-id defaults."
}

if (-not $envMap.ContainsKey("GCP_PROJECT_ID")) {
  $envMap["GCP_PROJECT_ID"] = $ProjectId
}

if (-not $envMap.ContainsKey("FIREBASE_PROJECT_ID")) {
  $envMap["FIREBASE_PROJECT_ID"] = $ProjectId
}

if (-not $envMap.ContainsKey("FIRESTORE_PROJECT_ID")) {
  $envMap["FIRESTORE_PROJECT_ID"] = $ProjectId
}

if (-not $envMap.ContainsKey("CLOUD_TASKS_PROJECT_ID")) {
  $envMap["CLOUD_TASKS_PROJECT_ID"] = $ProjectId
}

if (-not $envMap.ContainsKey("CLOUD_TASKS_LOCATION")) {
  $envMap["CLOUD_TASKS_LOCATION"] = $Region
}

if (-not $envMap.ContainsKey("INTERNAL_BASE_URL") -or [string]::IsNullOrWhiteSpace($envMap["INTERNAL_BASE_URL"])) {
  if ($envMap.ContainsKey("SELF_BASE_URL") -and -not [string]::IsNullOrWhiteSpace($envMap["SELF_BASE_URL"])) {
    $envMap["INTERNAL_BASE_URL"] = $envMap["SELF_BASE_URL"]
  }
}

if (-not $envMap.ContainsKey("SOLANA_RPC_URL") -or [string]::IsNullOrWhiteSpace($envMap["SOLANA_RPC_URL"])) {
  throw "SOLANA_RPC_URL is required. Set a devnet Solana RPC endpoint in .env before deploying."
}

if (-not $envMap.ContainsKey("SOLANA_MINT_KEY_SECRET") -or [string]::IsNullOrWhiteSpace($envMap["SOLANA_MINT_KEY_SECRET"])) {
  throw "SOLANA_MINT_KEY_SECRET is required. Set the Secret Manager version path in .env before deploying."
}

if ($envMap.ContainsKey("SOLANA_AUTO_AIRDROP_ENABLED") -and $envMap["SOLANA_AUTO_AIRDROP_ENABLED"].ToLower() -eq "true") {
  if (-not $envMap.ContainsKey("SOLANA_AIRDROP_RPC_URL") -or [string]::IsNullOrWhiteSpace($envMap["SOLANA_AIRDROP_RPC_URL"])) {
    throw "SOLANA_AIRDROP_RPC_URL is required when SOLANA_AUTO_AIRDROP_ENABLED=true."
  }

  if (-not $envMap.ContainsKey("SOLANA_MIN_FEE_PAYER_BALANCE_SOL") -or [string]::IsNullOrWhiteSpace($envMap["SOLANA_MIN_FEE_PAYER_BALANCE_SOL"])) {
    throw "SOLANA_MIN_FEE_PAYER_BALANCE_SOL is required when SOLANA_AUTO_AIRDROP_ENABLED=true."
  }

  if (-not $envMap.ContainsKey("SOLANA_AIRDROP_AMOUNT_SOL") -or [string]::IsNullOrWhiteSpace($envMap["SOLANA_AIRDROP_AMOUNT_SOL"])) {
    throw "SOLANA_AIRDROP_AMOUNT_SOL is required when SOLANA_AUTO_AIRDROP_ENABLED=true."
  }
}

if (-not $envMap.ContainsKey("SELF_BASE_URL") -or [string]::IsNullOrWhiteSpace($envMap["SELF_BASE_URL"])) {
  try {
    $selfUrl = (& $GCLOUD run services describe $ServiceName `
      --region $Region `
      --project $ProjectId `
      --format "value(status.url)")

    if ($selfUrl) {
      if ($selfUrl.EndsWith("/")) {
        $envMap["SELF_BASE_URL"] = $selfUrl.Substring(0, $selfUrl.Length - 1)
      } else {
        $envMap["SELF_BASE_URL"] = $selfUrl
      }

      Write-Ok "SELF_BASE_URL resolved from Cloud Run: $($envMap["SELF_BASE_URL"])"
    } else {
      Write-Warn "SELF_BASE_URL could not be resolved because service url is empty. Please set it in .env."
    }
  } catch {
    Write-Warn "Failed to resolve SELF_BASE_URL from Cloud Run. Please set it in .env."
  }
}

if (-not $envMap.ContainsKey("INTERNAL_BASE_URL") -or [string]::IsNullOrWhiteSpace($envMap["INTERNAL_BASE_URL"])) {
  if ($envMap.ContainsKey("SELF_BASE_URL") -and -not [string]::IsNullOrWhiteSpace($envMap["SELF_BASE_URL"])) {
    $envMap["INTERNAL_BASE_URL"] = $envMap["SELF_BASE_URL"]
    Write-Ok "INTERNAL_BASE_URL resolved from SELF_BASE_URL: $($envMap["INTERNAL_BASE_URL"])"
  }
}

if ($envMap.ContainsKey("CLOUD_TASKS_QUEUE_ID") -or $envMap.ContainsKey("CLOUD_TASKS_SERVICE_ACCOUNT")) {
  if (-not $envMap.ContainsKey("CLOUD_TASKS_QUEUE_ID") -or [string]::IsNullOrWhiteSpace($envMap["CLOUD_TASKS_QUEUE_ID"])) {
    throw "CLOUD_TASKS_QUEUE_ID is required when Cloud Tasks mint worker is enabled."
  }

  if (-not $envMap.ContainsKey("INTERNAL_BASE_URL") -or [string]::IsNullOrWhiteSpace($envMap["INTERNAL_BASE_URL"])) {
    throw "INTERNAL_BASE_URL is required when Cloud Tasks mint worker is enabled."
  }

  if (-not $envMap.ContainsKey("CLOUD_TASKS_SERVICE_ACCOUNT") -or [string]::IsNullOrWhiteSpace($envMap["CLOUD_TASKS_SERVICE_ACCOUNT"])) {
    throw "CLOUD_TASKS_SERVICE_ACCOUNT is required when Cloud Tasks mint worker is enabled."
  }
}

$envPairs = @()

foreach ($k in $envMap.Keys) {
  $v = $envMap[$k]
  if ($null -eq $v) { $v = "" }

  $envPairs += "$k=$v"
}

$envArg = [string]::Join(",", $envPairs)

Write-Step "Env vars to update: $envArg"

# ------------------------------------------------------------
# 6) Cloud Run へデプロイ
# ------------------------------------------------------------
Write-Step "Deploying to Cloud Run"

$removeEnvVars = @(
  # 既存の Windows ローカルパス系 env
  "GOOGLE_APPLICATION_CREDENTIALS",
  "FIRESTORE_CREDENTIALS_FILE",

  # 旧 Solana env
  "SOLANA_RPC_ENDPOINT",

  # 旧 Stripe env
  # Stripe secret は Secret Manager の stripe-secret-key を使用する
  "STRIPE_SECRET_KEY",
  "VITE_STRIPE_PUBLISHABLE_KEY",
  "STRIPE_PUBLIC_KEY"
)

$deployArgs = @(
  "run", "deploy", $ServiceName,
  "--image", $Image,
  "--region", $Region,
  "--platform", "managed",
  "--allow-unauthenticated",
  "--service-account", $RunServiceAccount,

  "--remove-env-vars", ([string]::Join(",", $removeEnvVars)),

  "--update-env-vars", $envArg,
  "--min-instances", "0",

  # 暫定的にmint時のRPC負荷を抑える。
  # 最終的にはmint workerをCloud Tasks化し、worker側だけ concurrency=1 にする。
  "--max-instances", "2",
  "--memory", "512Mi",
  "--cpu", "1",
  "--concurrency", "10",
  "--timeout", "60s",
  "--project", $ProjectId
)

& $GCLOUD @deployArgs

if ($LASTEXITCODE -ne 0) {
  throw "gcloud run deploy failed. exit code: $LASTEXITCODE"
}

Write-Ok "Cloud Run deployment finished: service '${ServiceName}'"
Write-Ok "Deployed with image: ${Image}"