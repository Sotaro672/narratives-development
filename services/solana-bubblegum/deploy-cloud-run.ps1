# services/solana-bubblegum/deploy-cloud-run.ps1

$ErrorActionPreference = "Stop"

$ProjectId = "narratives-development-26c2d"
$Region = "asia-northeast1"
$ServiceName = "solana-bubblegum"

$GoogleCloudProject = "narratives-development-26c2d"
$SolanaCluster = "devnet"
$SolanaDevnetAirdropRpcUrl = "https://api.devnet.solana.com"

$BubblegumReservePublicKey = "9VwuvEExAcBgfy8oupQEUrXTWd1NSdNmiQK7QSKED6Rs"
$BubblegumDevnetReserveTargetSol = "100"
$BubblegumDevnetAirdropSol = "5"
$BubblegumFeePayerTargetSol = "0.5"
$BubblegumReserveMinimumSol = "1"

$SolanaRpcSecretName = "bubblegum-solana-rpc-url"
$SolanaRpcSecretVersion = "1"

Write-Host "Deploying Cloud Run configuration..."
Write-Host "Project: $ProjectId"
Write-Host "Region: $Region"
Write-Host "Service: $ServiceName"

$EnvVars = @(
    "GOOGLE_CLOUD_PROJECT=$GoogleCloudProject",
    "SOLANA_CLUSTER=$SolanaCluster",
    "SOLANA_DEVNET_AIRDROP_RPC_URL=$SolanaDevnetAirdropRpcUrl",
    "BUBBLEGUM_RESERVE_PUBLIC_KEY=$BubblegumReservePublicKey",
    "BUBBLEGUM_DEVNET_RESERVE_TARGET_SOL=$BubblegumDevnetReserveTargetSol",
    "BUBBLEGUM_DEVNET_AIRDROP_SOL=$BubblegumDevnetAirdropSol",
    "BUBBLEGUM_FEE_PAYER_TARGET_SOL=$BubblegumFeePayerTargetSol",
    "BUBBLEGUM_RESERVE_MINIMUM_SOL=$BubblegumReserveMinimumSol"
) -join ","

$Secrets = @(
    "SOLANA_RPC_URL=${SolanaRpcSecretName}:${SolanaRpcSecretVersion}"
) -join ","

gcloud run services update $ServiceName `
    --project=$ProjectId `
    --region=$Region `
    --platform=managed `
    --update-env-vars=$EnvVars `
    --update-secrets=$Secrets

if ($LASTEXITCODE -ne 0) {
    throw "Cloud Run deployment failed."
}

Write-Host ""
Write-Host "Cloud Run deployment completed."
Write-Host ""
Write-Host "Current environment variables:"

gcloud run services describe $ServiceName `
    --project=$ProjectId `
    --region=$Region `
    --format="yaml(spec.template.spec.containers[0].env)"

if ($LASTEXITCODE -ne 0) {
    throw "Failed to read Cloud Run environment variables."
}

Write-Host ""
Write-Host "Service URL:"

gcloud run services describe $ServiceName `
    --project=$ProjectId `
    --region=$Region `
    --format="value(status.url)"

if ($LASTEXITCODE -ne 0) {
    throw "Failed to read Cloud Run service URL."
}