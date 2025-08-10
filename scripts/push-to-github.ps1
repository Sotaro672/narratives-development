# Push current state to GitHub repository

Write-Host "📤 Pushing Narratives Development to GitHub" -ForegroundColor Green
Write-Host "============================================" -ForegroundColor Green

# Check if we're in the correct directory
$currentPath = Get-Location
if ($currentPath.Path -notlike "*narratives-development") {
    Write-Host "❌ Please run this script from the narratives-development root directory" -ForegroundColor Red
    exit 1
}

# Check if git is initialized
if (!(Test-Path ".git")) {
    Write-Host "🔧 Initializing Git repository..." -ForegroundColor Blue
    git init
    git remote add origin https://github.com/Sotaro672/narratives-development.git
}

Write-Host ""
Write-Host "1️⃣ Checking Git status..." -ForegroundColor Blue
git status

Write-Host ""
Write-Host "2️⃣ Adding all files to staging..." -ForegroundColor Blue
git add .

Write-Host ""
Write-Host "3️⃣ Creating commit..." -ForegroundColor Blue
$commitMessage = Read-Host "Enter commit message (or press Enter for default)"
if ([string]::IsNullOrEmpty($commitMessage)) {
    $commitMessage = "feat: Complete Flutter mobile app setup with V2 embedding, Firebase integration, and deployment scripts

- ✅ Flutter mobile app with V2 Android embedding
- ✅ Firebase authentication integration 
- ✅ GraphQL client setup
- ✅ SNS UI with feed, posts, and navigation
- ✅ Web platform support for deployment
- ✅ Docker microservices architecture
- ✅ Firebase Hosting deployment configuration
- ✅ Automated setup and deployment scripts
- ✅ Android/iOS/Web multi-platform support"
}

git commit -m "$commitMessage"

Write-Host ""
Write-Host "4️⃣ Pushing to GitHub..." -ForegroundColor Blue
try {
    # Check if main branch exists remotely
    $branches = git ls-remote --heads origin
    if ($branches -like "*refs/heads/main*") {
        git push origin main
    } else {
        # Push to main branch and set upstream
        git branch -M main
        git push -u origin main
    }
    Write-Host "✅ Successfully pushed to GitHub!" -ForegroundColor Green
} catch {
    Write-Host "⚠️ Push failed. Trying to pull first..." -ForegroundColor Yellow
    try {
        git pull origin main --allow-unrelated-histories
        git push origin main
        Write-Host "✅ Successfully pushed after resolving conflicts!" -ForegroundColor Green
    } catch {
        Write-Host "❌ Push failed. Please check your GitHub credentials and repository access." -ForegroundColor Red
        Write-Host "You may need to set up authentication:" -ForegroundColor Yellow
        Write-Host "1. Personal Access Token: https://github.com/settings/tokens" -ForegroundColor White
        Write-Host "2. SSH Key: https://github.com/settings/keys" -ForegroundColor White
        exit 1
    }
}

Write-Host ""
Write-Host "🎉 Repository pushed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "Repository URL: https://github.com/Sotaro672/narratives-development" -ForegroundColor Cyan
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "• View repository: https://github.com/Sotaro672/narratives-development" -ForegroundColor White
Write-Host "• Set up GitHub Actions for CI/CD" -ForegroundColor White
Write-Host "• Configure branch protection rules" -ForegroundColor White
Write-Host "• Deploy to Firebase: .\scripts\deploy-mobile-to-firebase.ps1" -ForegroundColor White
