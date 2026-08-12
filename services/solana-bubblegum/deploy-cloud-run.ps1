# services/solana-bubblegum/deploy-cloud-run.ps1

$ErrorActionPreference = "Stop"

$ProjectId = "narratives-development-26c2d"
$Region = "asia-northeast1"
$ServiceName = "solana-bubblegum"

$GoogleCloudProject = "narratives-development-26c2d"
$SolanaCluster = "devnet"

$BubblegumFeePayerTargetSol = "0.5"
$BubblegumReserveMinimumSol = "1"

$SolanaRpcSecretName = "bubblegum-solana-rpc-url"
$SolanaRpcSecretVersion = "1"

$ScriptDirectory = Split-Path -Parent $MyInvocation.MyCommand.Path

Push-Location $ScriptDirectory

try {
    Write-Host ""
    Write-Host "========================================"
    Write-Host "Solana Bubblegum Cloud Run Deployment"
    Write-Host "========================================"
    Write-Host ""
    Write-Host "Project: $ProjectId"
    Write-Host "Region: $Region"
    Write-Host "Service: $ServiceName"
    Write-Host "Source: $ScriptDirectory"
    Write-Host ""

    if (-not (Test-Path ".\package.json")) {
        throw "package.json not found."
    }

    if (-not (Test-Path ".\src")) {
        throw "src directory not found."
    }

    Write-Host "Step 1/6: Running TypeScript typecheck..."

    npm run typecheck

    if ($LASTEXITCODE -ne 0) {
        throw "TypeScript typecheck failed."
    }

    Write-Host ""
    Write-Host "TypeScript typecheck completed."
    Write-Host ""

    Write-Host "Step 2/6: Building application..."

    npm run build

    if ($LASTEXITCODE -ne 0) {
        throw "Application build failed."
    }

    if (-not (Test-Path ".\dist\index.js")) {
        throw "Build completed but dist/index.js was not created."
    }

    Write-Host ""
    Write-Host "Application build completed."
    Write-Host ""

    $EnvVars = @(
        "GOOGLE_CLOUD_PROJECT=$GoogleCloudProject",
        "SOLANA_CLUSTER=$SolanaCluster",
        "BUBBLEGUM_FEE_PAYER_TARGET_SOL=$BubblegumFeePayerTargetSol",
        "BUBBLEGUM_RESERVE_MINIMUM_SOL=$BubblegumReserveMinimumSol"
    ) -join ","

    $RemovedEnvVars = @(
        "SOLANA_DEVNET_AIRDROP_RPC_URL",
        "BUBBLEGUM_RESERVE_PUBLIC_KEY",
        "BUBBLEGUM_DEVNET_RESERVE_TARGET_SOL",
        "BUBBLEGUM_DEVNET_AIRDROP_SOL"
    ) -join ","

    $Secrets = @(
        "SOLANA_RPC_URL=${SolanaRpcSecretName}:${SolanaRpcSecretVersion}"
    ) -join ","

    Write-Host "Step 3/6: Building source and deploying to Cloud Run..."
    Write-Host ""

    $DeployArgs = @(
        "run",
        "deploy",
        $ServiceName,
        "--source=.",
        "--project=$ProjectId",
        "--region=$Region",
        "--platform=managed",
        "--set-build-env-vars=GOOGLE_NODE_RUN_SCRIPTS=build",
        "--update-env-vars=$EnvVars",
        "--remove-env-vars=$RemovedEnvVars",
        "--update-secrets=$Secrets",
        "--quiet"
    )

    & gcloud @DeployArgs

    if ($LASTEXITCODE -ne 0) {
        throw "Cloud Run source deployment failed."
    }

    Write-Host ""
    Write-Host "Cloud Run source deployment completed."
    Write-Host ""

    Write-Host "Step 4/6: Reading latest revision..."

    $LatestRevision = & gcloud run services describe $ServiceName `
        --project=$ProjectId `
        --region=$Region `
        --format="value(status.latestReadyRevisionName)"

    if ($LASTEXITCODE -ne 0) {
        throw "Failed to read latest Cloud Run revision."
    }

    if ([string]::IsNullOrWhiteSpace($LatestRevision)) {
        throw "Latest Cloud Run revision is empty."
    }

    Write-Host ""
    Write-Host "Latest revision: $LatestRevision"
    Write-Host ""

    Write-Host "Step 5/6: Reading deployed container image..."

    $DeployedImage = & gcloud run revisions describe $LatestRevision `
        --project=$ProjectId `
        --region=$Region `
        --format="value(spec.containers[0].image)"

    if ($LASTEXITCODE -ne 0) {
        throw "Failed to read deployed container image."
    }

    Write-Host ""
    Write-Host "Deployed image:"
    Write-Host $DeployedImage
    Write-Host ""

    Write-Host "Current environment variables:"

    & gcloud run services describe $ServiceName `
        --project=$ProjectId `
        --region=$Region `
        --format="yaml(spec.template.spec.containers[0].env)"

    if ($LASTEXITCODE -ne 0) {
        throw "Failed to read Cloud Run environment variables."
    }

    Write-Host ""
    Write-Host "Step 6/6: Reading service URL..."

    $ServiceUrl = & gcloud run services describe $ServiceName `
        --project=$ProjectId `
        --region=$Region `
        --format="value(status.url)"

    if ($LASTEXITCODE -ne 0) {
        throw "Failed to read Cloud Run service URL."
    }

    if ([string]::IsNullOrWhiteSpace($ServiceUrl)) {
        throw "Cloud Run service URL is empty."
    }

    Write-Host ""
    Write-Host "========================================"
    Write-Host "Deployment completed successfully."
    Write-Host "========================================"
    Write-Host ""
    Write-Host "Service: $ServiceName"
    Write-Host "Revision: $LatestRevision"
    Write-Host "Image: $DeployedImage"
    Write-Host "URL: $ServiceUrl"
    Write-Host ""
}
finally {
    Pop-Location
}