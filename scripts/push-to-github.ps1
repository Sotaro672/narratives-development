# Push current state to GitHub

Write-Host "📤 Pushing Narratives SNS to GitHub" -ForegroundColor Green
Write-Host "====================================" -ForegroundColor Green

# Navigate to root directory
Set-Location $PSScriptRoot
Set-Location ".."

Write-Host "1️⃣ Checking Git status..." -ForegroundColor Blue

# Check if we're in a Git repository
if (!(Test-Path ".git")) {
    Write-Host "❌ Not a Git repository! Initializing..." -ForegroundColor Red
    git init
    Write-Host "✅ Git repository initialized" -ForegroundColor Green
}

# Check Git remote
$remotes = git remote -v 2>$null
if ([string]::IsNullOrEmpty($remotes)) {
    Write-Host "⚠️ No remote repository configured" -ForegroundColor Yellow
    Write-Host "Please configure your GitHub remote:" -ForegroundColor Yellow
    Write-Host "git remote add origin https://github.com/YOUR_USERNAME/narratives-development.git" -ForegroundColor White
    exit 1
}

Write-Host "✅ Git repository found" -ForegroundColor Green

Write-Host ""
Write-Host "2️⃣ Checking for changes..." -ForegroundColor Blue

# Check for changes
$status = git status --porcelain
if ([string]::IsNullOrEmpty($status)) {
    Write-Host "ℹ️ No changes to commit" -ForegroundColor Yellow
    
    # Try to push anyway in case there are unpushed commits
    Write-Host ""
    Write-Host "3️⃣ Checking for unpushed commits..." -ForegroundColor Blue
    
    $unpushed = git log origin/main..HEAD --oneline 2>$null
    if ([string]::IsNullOrEmpty($unpushed)) {
        Write-Host "✅ Everything is up to date!" -ForegroundColor Green
        exit 0
    } else {
        Write-Host "📤 Found unpushed commits, pushing..." -ForegroundColor Blue
        git push origin main
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✅ Successfully pushed to GitHub!" -ForegroundColor Green
        } else {
            Write-Host "❌ Failed to push to GitHub" -ForegroundColor Red
            exit 1
        }
        exit 0
    }
}

Write-Host "📝 Found changes to commit:" -ForegroundColor Green
git status --short

Write-Host ""
Write-Host "3️⃣ Creating comprehensive commit..." -ForegroundColor Blue

# Create .gitignore if it doesn't exist
if (!(Test-Path ".gitignore")) {
    Write-Host "📝 Creating .gitignore..." -ForegroundColor Blue
    
    $gitignore = @'
# Flutter/Dart
apps/sns/mobile/.dart_tool/
apps/sns/mobile/build/
apps/sns/mobile/.flutter-plugins
apps/sns/mobile/.flutter-plugins-dependencies
apps/sns/mobile/.packages
apps/sns/mobile/.pub-cache/
apps/sns/mobile/.pub/
apps/sns/mobile/pubspec.lock

# Go
services/sns-backend/vendor/
services/sns-backend/*.exe
services/sns-backend/*.dll
services/sns-backend/*.so
services/sns-backend/*.dylib
services/sns-backend/go.sum

# IDE
.vscode/
.idea/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Firebase
apps/sns/mobile/.firebase/
apps/sns/mobile/firebase-debug.log
apps/sns/mobile/firestore-debug.log

# Environment variables
.env
.env.local
.env.production

# Logs
*.log
npm-debug.log*
yarn-debug.log*
yarn-error.log*

# Node modules (if any)
node_modules/

# Temporary files
*.tmp
*.temp
'@

    $gitignore | Out-File -FilePath ".gitignore" -Encoding UTF8
    Write-Host "✅ .gitignore created" -ForegroundColor Green
}

# Add all changes
Write-Host "📋 Staging changes..." -ForegroundColor Blue
git add .

# Create detailed commit message
$timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
$commitMessage = @"
feat: Complete SNS application with Firebase integration

## 🚀 Major Features Added

### Flutter Mobile App
- ✅ Multi-tab navigation (Home, Explore, Post, Notifications, Profile)
- ✅ Post creation dialog with text and image URL support
- ✅ User profile management with comprehensive form
- ✅ Avatar information handling
- ✅ PostCard widget for displaying posts
- ✅ Provider pattern for state management
- ✅ Firebase integration ready
- ✅ Progressive Web App (PWA) support
- ✅ Japanese/English UI support

### Backend Services
- ✅ Go backend with GraphQL API structure
- ✅ Firebase Firestore integration
- ✅ Cloud Storage support
- ✅ User and Post management entities

### Infrastructure
- ✅ Firebase hosting configuration
- ✅ PWA manifest with icons and shortcuts
- ✅ Deployment scripts for automation
- ✅ Web asset generation scripts

### File Structure
```
apps/sns/mobile/
├── lib/
│   ├── main.dart (comprehensive app entry point)
│   ├── screens/home/home_screen.dart (main navigation)
│   ├── widgets/
│   │   ├── create_post_dialog.dart
│   │   └── post_card.dart
│   ├── providers/
│   │   ├── post_provider.dart
│   │   └── user_provider.dart
│   └── services/
│       ├── firestore_service.dart
│       └── gcs_service.dart
├── web/
│   ├── index.html (Firebase optimized)
│   ├── manifest.json (PWA configuration)
│   └── icons/ (app icons)
└── pubspec.yaml (dependencies)

services/sns-backend/
├── go.mod (Go dependencies)
└── (GraphQL API structure)

scripts/
├── deploy-current-state.ps1
├── push-to-github.ps1
└── fix-and-deploy.ps1
```

## 🔧 Technical Improvements
- Firebase Hosting deployment ready
- Web renderer optimizations
- Service worker configuration
- Responsive design implementation
- Error handling and validation
- State management with Provider pattern

## 🌐 Deployment
- Firebase project: narratives-development-26c2d
- Live URLs ready for deployment
- PWA installable on mobile devices

## 📱 User Experience
- Clean, modern UI design
- Intuitive navigation
- Real-time post creation
- Profile customization
- Mobile-first responsive design

Committed at: $timestamp
"@

# Commit changes
Write-Host "💾 Committing changes..." -ForegroundColor Blue
git commit -m $commitMessage

if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Failed to commit changes" -ForegroundColor Red
    exit 1
}

Write-Host "✅ Changes committed successfully" -ForegroundColor Green

Write-Host ""
Write-Host "4️⃣ Pushing to GitHub..." -ForegroundColor Blue

# Get current branch
$branch = git rev-parse --abbrev-ref HEAD

# Push to GitHub
git push origin $branch

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "🎉 Successfully pushed to GitHub!" -ForegroundColor Green
    Write-Host "=================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "📊 Summary of changes pushed:" -ForegroundColor Yellow
    Write-Host "   ✅ Complete Flutter SNS application" -ForegroundColor Green
    Write-Host "   ✅ Firebase integration and deployment scripts" -ForegroundColor Green
    Write-Host "   ✅ Go backend structure with GraphQL" -ForegroundColor Green
    Write-Host "   ✅ PWA configuration and web assets" -ForegroundColor Green
    Write-Host "   ✅ Comprehensive documentation" -ForegroundColor Green
    
    Write-Host ""
    Write-Host "🔗 Repository information:" -ForegroundColor Cyan
    $remoteUrl = git config --get remote.origin.url
    Write-Host "   📦 Remote URL: $remoteUrl" -ForegroundColor White
    Write-Host "   🌿 Branch: $branch" -ForegroundColor White
    
    $lastCommit = git log -1 --format="%h - %s"
    Write-Host "   📝 Last commit: $lastCommit" -ForegroundColor White
    
} else {
    Write-Host ""
    Write-Host "❌ Failed to push to GitHub!" -ForegroundColor Red
    Write-Host ""
    Write-Host "🔧 Troubleshooting:" -ForegroundColor Yellow
    Write-Host "   1. Check your GitHub authentication" -ForegroundColor White
    Write-Host "   2. Verify remote repository URL" -ForegroundColor White
    Write-Host "   3. Check branch permissions" -ForegroundColor White
    Write-Host "   4. Try: git push --set-upstream origin $branch" -ForegroundColor White
    exit 1
}

Write-Host ""
Write-Host "✅ GitHub push completed!" -ForegroundColor Green
Write-Host "Current time: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Gray
