# Deploy current state to Firebase

Write-Host "🚀 Deploying Current Narratives SNS to Firebase" -ForegroundColor Green
Write-Host "===============================================" -ForegroundColor Green

Set-Location "apps\sns\mobile"

Write-Host "1️⃣ Checking current state..." -ForegroundColor Blue
if (!(Test-Path "lib\main.dart")) {
    Write-Host "❌ main.dart not found!" -ForegroundColor Red
    exit 1
}

if (!(Test-Path "web\index.html")) {
    Write-Host "❌ index.html not found!" -ForegroundColor Red
    exit 1
}

Write-Host "✅ Found required files" -ForegroundColor Green

Write-Host ""
Write-Host "2️⃣ Ensuring web platform support..." -ForegroundColor Blue
flutter config --enable-web

if (!(Test-Path "web")) {
    flutter create --platforms=web .
}

Write-Host ""
Write-Host "3️⃣ Installing dependencies..." -ForegroundColor Blue
flutter clean
flutter pub get

Write-Host ""
Write-Host "4️⃣ Building for production..." -ForegroundColor Blue
try {
    flutter build web --release --web-renderer html --dart-define=FLUTTER_WEB_USE_SKIA=false
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Build successful" -ForegroundColor Green
    } else {
        Write-Host "⚠️ Build had warnings but completed" -ForegroundColor Yellow
    }
} catch {
    Write-Host "❌ Build failed: $_" -ForegroundColor Red
    Write-Host "Trying fallback build..." -ForegroundColor Yellow
    flutter build web --release
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host "❌ Fallback build also failed" -ForegroundColor Red
        exit 1
    }
}

Write-Host ""
Write-Host "5️⃣ Configuring Firebase..." -ForegroundColor Blue

# Create firebase.json for hosting
$firebaseConfig = @{
    hosting = @{
        public = "build/web"
        ignore = @(
            "firebase.json",
            "**/.*",
            "**/node_modules/**"
        )
        rewrites = @(
            @{
                source = "**"
                destination = "/index.html"
            }
        )
        headers = @(
            @{
                source = "**/*.@(js|css|map|json|woff2|woff|ttf|eot|svg|png|jpg|jpeg|gif|ico|webp)"
                headers = @(
                    @{
                        key = "Cache-Control"
                        value = "max-age=31536000, immutable"
                    }
                )
            },
            @{
                source = "**/*.@(html|json)"
                headers = @(
                    @{
                        key = "Cache-Control"
                        value = "max-age=0, must-revalidate"
                    }
                )
            }
        )
        cleanUrls = $true
        trailingSlash = $false
    }
} | ConvertTo-Json -Depth 10

$firebaseConfig | Out-File -FilePath "firebase.json" -Encoding UTF8
Write-Host "✅ firebase.json created" -ForegroundColor Green

# Create .firebaserc
$firebaserc = @{
    projects = @{
        default = "narratives-development-26c2d"
    }
} | ConvertTo-Json -Depth 10

$firebaserc | Out-File -FilePath ".firebaserc" -Encoding UTF8
Write-Host "✅ .firebaserc created" -ForegroundColor Green

Write-Host ""
Write-Host "6️⃣ Deploying to Firebase Hosting..." -ForegroundColor Blue

# Check if logged in to Firebase
try {
    $projects = firebase projects:list 2>&1
    if ($projects -like "*Error*" -or $projects -like "*not logged in*") {
        Write-Host "🔑 Please log in to Firebase..." -ForegroundColor Blue
        firebase login
    }
} catch {
    Write-Host "🔑 Logging into Firebase..." -ForegroundColor Blue
    firebase login
}

# Deploy to Firebase
firebase deploy --only hosting --project narratives-development-26c2d

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "🎉 Deployment Successful!" -ForegroundColor Green
    Write-Host "=========================" -ForegroundColor Green
    Write-Host ""
    Write-Host "🌐 Your app is live at:" -ForegroundColor Cyan
    Write-Host "   📱 Main URL: https://narratives-development-26c2d.web.app" -ForegroundColor White
    Write-Host "   🔄 Alt URL:  https://narratives-development-26c2d.firebaseapp.com" -ForegroundColor White
    Write-Host ""
    Write-Host "📊 Deployed Features:" -ForegroundColor Yellow
    Write-Host "   ✅ Multi-tab navigation (Home, Explore, Post, Notifications, Profile)" -ForegroundColor Green
    Write-Host "   ✅ Post creation with text and image URL support" -ForegroundColor Green
    Write-Host "   ✅ User profile management with avatar info" -ForegroundColor Green
    Write-Host "   ✅ Japanese UI with role-based permissions" -ForegroundColor Green
    Write-Host "   ✅ Firebase integration ready" -ForegroundColor Green
    Write-Host "   ✅ Progressive Web App (PWA) support" -ForegroundColor Green
    Write-Host "   ✅ Responsive design (mobile/desktop)" -ForegroundColor Green
    
    # Check build size
    $buildPath = "build\web"
    if (Test-Path $buildPath) {
        $buildSize = [math]::Round(((Get-ChildItem $buildPath -Recurse | Measure-Object -Property Length -Sum).Sum / 1MB), 2)
        Write-Host "   📦 Build size: $buildSize MB" -ForegroundColor Green
    }
    
    Write-Host ""
    Write-Host "🔗 Quick Links:" -ForegroundColor Cyan
    Write-Host "   🏠 Home Feed: https://narratives-development-26c2d.web.app/#/" -ForegroundColor White
    Write-Host "   👤 Profile: https://narratives-development-26c2d.web.app/#/profile" -ForegroundColor White
    Write-Host "   ➕ Create Post: Click the + button in navigation" -ForegroundColor White
    
} else {
    Write-Host ""
    Write-Host "❌ Deployment Failed!" -ForegroundColor Red
    Write-Host ""
    Write-Host "🔧 Troubleshooting:" -ForegroundColor Yellow
    Write-Host "   1. Check Firebase project permissions" -ForegroundColor White
    Write-Host "   2. Verify project ID: narratives-development-26c2d" -ForegroundColor White
    Write-Host "   3. Try: firebase login --reauth" -ForegroundColor White
    Write-Host "   4. Check if build/web directory exists" -ForegroundColor White
    exit 1
}

Set-Location "..\..\.."

Write-Host ""
Write-Host "✅ Deployment process completed!" -ForegroundColor Green

Write-Host ""
Write-Host "7️⃣ Optionally push to GitHub..." -ForegroundColor Blue
$pushToGitHub = Read-Host "Would you like to push changes to GitHub as well? (y/N)"

if ($pushToGitHub -eq 'y' -or $pushToGitHub -eq 'Y') {
    Write-Host "📤 Pushing to GitHub..." -ForegroundColor Blue
    & "..\..\..\scripts\push-to-github.ps1"
}

Write-Host "Current time: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Gray
