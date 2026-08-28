param(
  # Example explicit image:
  # asia-northeast1-docker.pkg.dev/<PROJECT>/<REPO>/<SERVICE>:<TAG>
  [string]$Image,

  # Region / Cloud Run service name
  [string]$Region = "asia-northeast1",
  [string]$ServiceName = "narratives-backend",

  # Artifact Registry repository name
  [string]$RepoName = "narratives-backend",

  # Mint / internal worker Cloud Tasks queue
  [string]$CloudTasksQueueID = "mint-product-tasks"
)

$ErrorActionPreference = "Stop"

function Write-Step($msg) {
  Write-Host "== $msg ==" -ForegroundColor Cyan
}

function Write-Ok($msg) {
  Write-Host "OK: $msg" -ForegroundColor Green
}

function Write-Warn($msg) {
  Write-Host "!! $msg ==" -ForegroundColor Yellow
}

function Normalize-EnvValue([string]$Value) {
  if ($null -eq $Value) {
    return ""
  }

  $Normalized = $Value

  if (
    ($Normalized.StartsWith('"') -and $Normalized.EndsWith('"')) -or
    ($Normalized.StartsWith("'") -and $Normalized.EndsWith("'"))
  ) {
    $Normalized = $Normalized.Substring(
      1,
      $Normalized.Length - 2
    )
  }

  return $Normalized
}

function Read-EnvFile([string]$Path) {
  $Map = @{}

  foreach ($Line in Get-Content $Path) {
    if ($null -eq $Line) {
      continue
    }

    if ($Line -eq "") {
      continue
    }

    if ($Line.StartsWith("#")) {
      continue
    }

    $SeparatorIndex = $Line.IndexOf("=")

    if ($SeparatorIndex -lt 1) {
      continue
    }

    $Key = $Line.Substring(
      0,
      $SeparatorIndex
    ).Trim()

    $Value = $Line.Substring(
      $SeparatorIndex + 1
    )

    $Map[$Key] = Normalize-EnvValue $Value
  }

  return $Map
}

function Invoke-CloudBuildOrThrow {
  param(
    [Parameter(Mandatory=$true)]
    [string]$Image
  )

  Write-Step "Running Cloud Build"
  Write-Step "Cloud Build image: $Image"

  $CloudBuildIgnoreFile =
    Join-Path $SourceDir ".gcloudignore"

  if (-not (Test-Path $CloudBuildIgnoreFile)) {
    throw ".gcloudignore not found: $CloudBuildIgnoreFile"
  }

  Write-Step "Cloud Build ignore file: $CloudBuildIgnoreFile"

  Push-Location $SourceDir

  try {
    & $GCLOUD builds submit `
      "." `
      --tag "$Image" `
      --project "$ProjectId" `
      --ignore-file="$CloudBuildIgnoreFile"

    if ($LASTEXITCODE -ne 0) {
      throw "Cloud Build failed. exit code: $LASTEXITCODE"
    }
  }
  finally {
    Pop-Location
  }

  Write-Ok "Image build & push completed by Cloud Build"
}

function Ensure-CloudTasksQueueRunning {
  param(
    [Parameter(Mandatory=$true)]
    [string]$QueueID,

    [Parameter(Mandatory=$true)]
    [string]$Location,

    [Parameter(Mandatory=$true)]
    [string]$ProjectID
  )

  if ([string]::IsNullOrWhiteSpace($QueueID)) {
    throw "Cloud Tasks queue ID is empty."
  }

  if ([string]::IsNullOrWhiteSpace($Location)) {
    throw "Cloud Tasks location is empty."
  }

  if ([string]::IsNullOrWhiteSpace($ProjectID)) {
    throw "Cloud Tasks project ID is empty."
  }

  Write-Step "Checking Cloud Tasks queue state: $QueueID"

  $QueueState = (
    & $GCLOUD tasks queues describe $QueueID `
      --location=$Location `
      --project=$ProjectID `
      --format="value(state)"
  )

  if ($LASTEXITCODE -ne 0) {
    throw "Failed to describe Cloud Tasks queue '$QueueID'."
  }

  $QueueState = "$QueueState".Trim().ToUpperInvariant()

  if ([string]::IsNullOrWhiteSpace($QueueState)) {
    throw "Cloud Tasks queue '$QueueID' returned an empty state."
  }

  Write-Step "Cloud Tasks queue state: $QueueState"

  if ($QueueState -eq "RUNNING") {
    Write-Ok "Cloud Tasks queue is running: $QueueID"
    return
  }

  if ($QueueState -ne "PAUSED") {
    throw "Cloud Tasks queue '$QueueID' is not runnable. state=$QueueState"
  }

  Write-Warn "Cloud Tasks queue '$QueueID' is paused. Resuming it."

  & $GCLOUD tasks queues resume $QueueID `
    --location=$Location `
    --project=$ProjectID

  if ($LASTEXITCODE -ne 0) {
    throw "Failed to resume Cloud Tasks queue '$QueueID'."
  }

  $QueueStateAfterResume = (
    & $GCLOUD tasks queues describe $QueueID `
      --location=$Location `
      --project=$ProjectID `
      --format="value(state)"
  )

  if ($LASTEXITCODE -ne 0) {
    throw "Failed to verify Cloud Tasks queue '$QueueID' after resume."
  }

  $QueueStateAfterResume =
    "$QueueStateAfterResume".Trim().ToUpperInvariant()

  if ($QueueStateAfterResume -ne "RUNNING") {
    throw "Cloud Tasks queue '$QueueID' did not become RUNNING after resume. state=$QueueStateAfterResume"
  }

  Write-Ok "Cloud Tasks queue resumed: $QueueID"
}

# ------------------------------------------------------------
# 0) gcloud environment
# ------------------------------------------------------------

Write-Step "Starting deploy-backend.ps1"

$env:CLOUDSDK_CORE_DISABLE_PROMPTS = "1"
$env:CLOUDSDK_COMPONENT_MANAGER_DISABLE_UPDATE_CHECK = "1"

$GCLOUD = (
  Get-Command gcloud.cmd -ErrorAction Stop
).Source

Write-Step "Using gcloud.cmd: $GCLOUD"

$ProjectId = (
  & $GCLOUD config get-value project
)

$ConfiguredAccount = (
  & $GCLOUD config get-value account
)

if ([string]::IsNullOrWhiteSpace($ProjectId)) {
  throw "gcloud config project is not set. Example: gcloud config set project <PROJECT_ID>"
}

if ([string]::IsNullOrWhiteSpace($ConfiguredAccount)) {
  throw "gcloud active account is not set. Example: gcloud auth login"
}

Write-Ok "gcloud project: $ProjectId"
Write-Ok "gcloud account: $ConfiguredAccount"

$RunServiceAccount =
  "narratives-backend-sa@$ProjectId.iam.gserviceaccount.com"

$ScriptDir =
  Split-Path -Parent $MyInvocation.MyCommand.Path

$SourceDir = $ScriptDir

# ------------------------------------------------------------
# 1) Go build check
# ------------------------------------------------------------

$MainGo =
  Join-Path $SourceDir "cmd\api\main.go"

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
}
finally {
  Pop-Location
}

Write-Ok "go build succeeded"

# ------------------------------------------------------------
# 2) Artifact Registry
# ------------------------------------------------------------

Write-Step "Ensuring Artifact Registry repository: $RepoName"

& $GCLOUD artifacts repositories describe $RepoName `
  --location=$Region `
  --project=$ProjectId | Out-Null

$RepositoryExists =
  $LASTEXITCODE -eq 0

if (-not $RepositoryExists) {
  Write-Warn "Artifact Registry repository '$RepoName' was not found. Creating it."

  & $GCLOUD artifacts repositories create $RepoName `
    --repository-format=docker `
    --location=$Region `
    --description="Backend images for $ServiceName" `
    --project=$ProjectId | Out-Null

  if ($LASTEXITCODE -ne 0) {
    throw "Failed to create Artifact Registry repository '$RepoName'."
  }

  Write-Ok "Repository created: $RepoName"
}
else {
  Write-Ok "Repository exists: $RepoName"
}

# ------------------------------------------------------------
# 3) Container image
# ------------------------------------------------------------

if ([string]::IsNullOrWhiteSpace($Image)) {
  $RegistryHost =
    "$Region-docker.pkg.dev"

  $Tag =
    Get-Date -Format "yyyyMMdd-HHmmss"

  $Image =
    "$RegistryHost/$ProjectId/$RepoName/${ServiceName}:$Tag"

  Write-Step "Generated image: $Image"
}
else {
  Write-Step "Using specified image: $Image"
}

# ------------------------------------------------------------
# 4) Cloud Build
# ------------------------------------------------------------

Invoke-CloudBuildOrThrow -Image $Image

# ------------------------------------------------------------
# 5) Read .env
#
# .env から読むのは、GCP の project / region / service account
# から自動生成できない値だけにします。
# ------------------------------------------------------------

Write-Step "Collecting application environment variables"

$AllowedKeys = @(
  # Firebase Storage
  "FIREBASE_STORAGE_BUCKET",

  # Resend
  "RESEND_FROM",
  "RESEND_CONTACT_ADMIN_TO",
  "CONSOLE_BASE_URL",

  # Solana / Bubblegum V2
  "SOLANA_BUBBLEGUM_SERVICE_URL",
  "SOLANA_BUBBLEGUM_MINT_AUTHORITY_PUBLIC_KEY",

  # Arweave / Irys
  "ARWEAVE_BASE_URL",

  # Existing Cloud Run URL fallback
  "SELF_BASE_URL",

  # Stripe webhook
  "STRIPE_WEBHOOK_SECRET",

  # Mall
  "MALL_FRONTEND_BASE_URL",
  "MALL_AUTO_CREATE_STRIPE_TEST_PAYMENT_METHOD",

  # Settlement
  "SETTLEMENT_PLATFORM_FEE_RATE",
  "SETTLEMENT_PLATFORM_FEE_BASE"
)

$envMap = @{}

$EnvFile =
  Join-Path $SourceDir ".env"

if (Test-Path $EnvFile) {
  Write-Ok "Found .env: $EnvFile"

  $FileMap =
    Read-EnvFile $EnvFile

  foreach ($Key in $AllowedKeys) {
    if ($FileMap.ContainsKey($Key)) {
      $envMap[$Key] = $FileMap[$Key]
    }
  }
}
else {
  Write-Warn ".env file not found: $EnvFile"
}

# ------------------------------------------------------------
# 6) Project values
#
# Project ID は gcloud config を唯一の source とします。
# ------------------------------------------------------------

$envMap["GOOGLE_CLOUD_PROJECT"] = $ProjectId
$envMap["GCP_PROJECT_ID"] = $ProjectId
$envMap["FIREBASE_PROJECT_ID"] = $ProjectId
$envMap["FIRESTORE_PROJECT_ID"] = $ProjectId

# ------------------------------------------------------------
# 7) Required application settings
# ------------------------------------------------------------

if (
  -not $envMap.ContainsKey("FIREBASE_STORAGE_BUCKET") -or
  [string]::IsNullOrWhiteSpace(
    $envMap["FIREBASE_STORAGE_BUCKET"]
  )
) {
  throw "FIREBASE_STORAGE_BUCKET is required."
}

if (
  -not $envMap.ContainsKey("SOLANA_BUBBLEGUM_SERVICE_URL") -or
  [string]::IsNullOrWhiteSpace(
    $envMap["SOLANA_BUBBLEGUM_SERVICE_URL"]
  )
) {
  throw "SOLANA_BUBBLEGUM_SERVICE_URL is required."
}

if (
  -not $envMap.ContainsKey(
    "SOLANA_BUBBLEGUM_MINT_AUTHORITY_PUBLIC_KEY"
  ) -or
  [string]::IsNullOrWhiteSpace(
    $envMap[
      "SOLANA_BUBBLEGUM_MINT_AUTHORITY_PUBLIC_KEY"
    ]
  )
) {
  throw "SOLANA_BUBBLEGUM_MINT_AUTHORITY_PUBLIC_KEY is required."
}

if (
  -not $envMap.ContainsKey("MALL_FRONTEND_BASE_URL") -or
  [string]::IsNullOrWhiteSpace(
    $envMap["MALL_FRONTEND_BASE_URL"]
  )
) {
  throw "MALL_FRONTEND_BASE_URL is required."
}

$MallFrontendBaseURL =
  $envMap["MALL_FRONTEND_BASE_URL"].TrimEnd("/")

$MallFrontendURI = $null

if (
  -not [System.Uri]::TryCreate(
    $MallFrontendBaseURL,
    [System.UriKind]::Absolute,
    [ref]$MallFrontendURI
  ) -or
  (
    $MallFrontendURI.Scheme -ne "https" -and
    $MallFrontendURI.Scheme -ne "http"
  ) -or
  [string]::IsNullOrWhiteSpace($MallFrontendURI.Host)
) {
  throw "MALL_FRONTEND_BASE_URL must be an absolute http/https URL."
}

if (
  -not [string]::IsNullOrWhiteSpace($MallFrontendURI.AbsolutePath) -and
  $MallFrontendURI.AbsolutePath -ne "/"
) {
  throw "MALL_FRONTEND_BASE_URL must contain only the origin and must not contain a path."
}

if (-not [string]::IsNullOrWhiteSpace($MallFrontendURI.Query)) {
  throw "MALL_FRONTEND_BASE_URL must not contain a query string."
}

if (-not [string]::IsNullOrWhiteSpace($MallFrontendURI.Fragment)) {
  throw "MALL_FRONTEND_BASE_URL must not contain a fragment."
}

$envMap["MALL_FRONTEND_BASE_URL"] =
  $MallFrontendBaseURL

if (
  -not $envMap.ContainsKey(
    "MALL_AUTO_CREATE_STRIPE_TEST_PAYMENT_METHOD"
  ) -or
  [string]::IsNullOrWhiteSpace(
    $envMap[
      "MALL_AUTO_CREATE_STRIPE_TEST_PAYMENT_METHOD"
    ]
  )
) {
  $envMap[
    "MALL_AUTO_CREATE_STRIPE_TEST_PAYMENT_METHOD"
  ] = "false"
}

# Bubblegum Cloud Run の ID Token audience は
# service URL と同一にします。
$envMap["SOLANA_BUBBLEGUM_SERVICE_AUDIENCE"] =
  $envMap["SOLANA_BUBBLEGUM_SERVICE_URL"]

# ------------------------------------------------------------
# 8) Resolve backend Cloud Run URL
# ------------------------------------------------------------

$ResolvedBackendURL = ""

try {
  $ExistingServiceURL = (
    & $GCLOUD run services describe $ServiceName `
      --region=$Region `
      --project=$ProjectId `
      --format="value(status.url)"
  )

  if (
    $LASTEXITCODE -eq 0 -and
    -not [string]::IsNullOrWhiteSpace($ExistingServiceURL)
  ) {
    $ResolvedBackendURL =
      $ExistingServiceURL.TrimEnd("/")

    Write-Ok "Backend URL resolved from Cloud Run: $ResolvedBackendURL"
  }
}
catch {
  Write-Warn "Could not resolve existing Cloud Run service URL."
}

if (
  [string]::IsNullOrWhiteSpace($ResolvedBackendURL) -and
  $envMap.ContainsKey("SELF_BASE_URL") -and
  -not [string]::IsNullOrWhiteSpace(
    $envMap["SELF_BASE_URL"]
  )
) {
  $ResolvedBackendURL =
    $envMap["SELF_BASE_URL"].TrimEnd("/")

  Write-Ok "Backend URL resolved from SELF_BASE_URL: $ResolvedBackendURL"
}

if ([string]::IsNullOrWhiteSpace($ResolvedBackendURL)) {
  throw "Backend Cloud Run URL could not be resolved. Set SELF_BASE_URL in .env for the first deployment."
}

$envMap["SELF_BASE_URL"] =
  $ResolvedBackendURL

$envMap["INTERNAL_BASE_URL"] =
  $ResolvedBackendURL

# ------------------------------------------------------------
# 9) Cloud Tasks
#
# project / location / service account / audience は
# deployment settings から自動生成します。
# ------------------------------------------------------------

if ([string]::IsNullOrWhiteSpace($CloudTasksQueueID)) {
  throw "CloudTasksQueueID is empty."
}

$envMap["CLOUD_TASKS_PROJECT_ID"] =
  $ProjectId

$envMap["CLOUD_TASKS_LOCATION"] =
  $Region

$envMap["CLOUD_TASKS_QUEUE_ID"] =
  $CloudTasksQueueID

$envMap["CLOUD_TASKS_SERVICE_ACCOUNT"] =
  $RunServiceAccount

$envMap["CLOUD_TASKS_AUDIENCE"] =
  $ResolvedBackendURL

# MINT_TASK_DISPATCH_DELAY_SECONDS は設定しません。
# runtime の default 0 = 即時実行を使用します。

# ------------------------------------------------------------
# 10) List Save Operation Retry
#
# 現状は共通 Cloud Tasks queue を再利用します。
# .env に重複設定は持たせません。
# ------------------------------------------------------------

$envMap["LIST_SAVE_OPERATION_QUEUE_PROJECT_ID"] =
  $ProjectId

$envMap["LIST_SAVE_OPERATION_QUEUE_LOCATION"] =
  $Region

$envMap["LIST_SAVE_OPERATION_QUEUE_ID"] =
  $CloudTasksQueueID

$envMap["LIST_SAVE_OPERATION_QUEUE_TARGET_BASE_URL"] =
  $ResolvedBackendURL

$envMap[
  "LIST_SAVE_OPERATION_QUEUE_SERVICE_ACCOUNT_EMAIL"
] =
  $RunServiceAccount

$envMap["LIST_SAVE_OPERATION_QUEUE_OIDC_AUDIENCE"] =
  $ResolvedBackendURL

# ------------------------------------------------------------
# 11) Token Blueprint Create Operation
#
# 現状は共通 Cloud Tasks queue を再利用します。
# .env に重複設定は持たせません。
# ------------------------------------------------------------

$envMap["TOKEN_BLUEPRINT_CREATE_OPERATION_QUEUE_PROJECT_ID"] =
  $ProjectId

$envMap["TOKEN_BLUEPRINT_CREATE_OPERATION_QUEUE_LOCATION"] =
  $Region

$envMap["TOKEN_BLUEPRINT_CREATE_OPERATION_QUEUE_ID"] =
  $CloudTasksQueueID

$envMap["TOKEN_BLUEPRINT_CREATE_OPERATION_QUEUE_TARGET_BASE_URL"] =
  $ResolvedBackendURL

$envMap[
  "TOKEN_BLUEPRINT_CREATE_OPERATION_QUEUE_SERVICE_ACCOUNT_EMAIL"
] =
  $RunServiceAccount

$envMap["TOKEN_BLUEPRINT_CREATE_OPERATION_QUEUE_OIDC_AUDIENCE"] =
  $ResolvedBackendURL

# ------------------------------------------------------------
# 12) Build Cloud Run env argument
# ------------------------------------------------------------

$envPairs = @()

foreach ($Key in $envMap.Keys) {
  $Value = $envMap[$Key]

  if ($null -eq $Value) {
    $Value = ""
  }

  $envPairs += "$Key=$Value"
}

$envArg =
  [string]::Join(
    ",",
    $envPairs
  )

$envKeysForLog =
  [string]::Join(
    ",",
    ($envMap.Keys | Sort-Object)
  )

Write-Step "Env vars to update: $envKeysForLog"

# ------------------------------------------------------------
# 13) Remove obsolete Cloud Run environment variables
# ------------------------------------------------------------

$removeEnvVars = @(
  # Local credential paths must never be used on Cloud Run
  "GOOGLE_APPLICATION_CREDENTIALS",
  "FIRESTORE_CREDENTIALS_FILE",

  # Old Solana RPC / signer / fee payer / reserve settings
  "SOLANA_RPC_ENDPOINT",
  "SOLANA_RPC_URL",
  "SOLANA_AIRDROP_RPC_URL",
  "SOLANA_AUTO_AIRDROP_ENABLED",
  "SOLANA_AIRDROP_AMOUNT_SOL",
  "SOLANA_MIN_FEE_PAYER_BALANCE_SOL",
  "SOLANA_MINT_KEY_SECRET",
  "SOLANA_SELLER_FEE_BPS",
  "SOLANA_AUTO_TOP_UP_ENABLED",
  "SOLANA_FEE_PAYER_TARGET_BALANCE_SOL",
  "SOLANA_RESERVE_KEY_SECRET",
  "SOLANA_RESERVE_MIN_REMAINING_SOL",
  "SOLANA_RESERVE_TX_FEE_BUFFER_SOL",

  # Bubblegum service owns these responsibilities
  "BUBBLEGUM_FEE_PAYER_TARGET_SOL",
  "BUBBLEGUM_RESERVE_MINIMUM_SOL",
  "BUBBLEGUM_RESERVE_PUBLIC_KEY",

  # Dispatch delay defaults to zero in runtime
  "MINT_TASK_DISPATCH_DELAY_SECONDS",

  # Old Stripe settings
  "STRIPE_SECRET_KEY",
  "VITE_STRIPE_PUBLISHABLE_KEY",
  "STRIPE_PUBLIC_KEY"
)

# ------------------------------------------------------------
# 14) Deploy Cloud Run
# ------------------------------------------------------------

Write-Step "Deploying to Cloud Run"

$deployArgs = @(
  "run",
  "deploy",
  $ServiceName,

  "--image",
  $Image,

  "--region",
  $Region,

  "--platform",
  "managed",

  "--allow-unauthenticated",

  "--service-account",
  $RunServiceAccount,

  "--remove-env-vars",
  ([string]::Join(",", $removeEnvVars)),

  "--update-env-vars",
  $envArg,

  "--min-instances",
  "0",

  "--max-instances",
  "2",

  "--memory",
  "512Mi",

  "--cpu",
  "1",

  "--concurrency",
  "10",

  "--timeout",
  "60s",

  "--project",
  $ProjectId
)

& $GCLOUD @deployArgs

if ($LASTEXITCODE -ne 0) {
  throw "gcloud run deploy failed. exit code: $LASTEXITCODE"
}

Write-Ok "Cloud Run deployment finished: service '$ServiceName'"

# ------------------------------------------------------------
# 15) Ensure Cloud Tasks queue is running
#
# Cloud Tasks queue が PAUSED の場合、
# CreateTask 自体は成功しても worker は実行されません。
#
# Cloud Run の新revisionを正常にデプロイした後で queue を再開し、
# 保留中の mint task を新revisionへ流します。
# ------------------------------------------------------------

Ensure-CloudTasksQueueRunning `
  -QueueID $CloudTasksQueueID `
  -Location $Region `
  -ProjectID $ProjectId

# ------------------------------------------------------------
# 16) Deployment summary
# ------------------------------------------------------------

Write-Ok "Deployed image: $Image"
Write-Ok "Backend URL: $ResolvedBackendURL"
Write-Ok "Cloud Tasks queue: $CloudTasksQueueID"