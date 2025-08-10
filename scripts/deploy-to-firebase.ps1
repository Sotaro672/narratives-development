# Deploy Narratives SNS to Firebase Hosting

Write-Host "🚀 Deploying Narratives SNS to Firebase Hosting" -ForegroundColor Green
Write-Host "================================================" -ForegroundColor Green

# Check prerequisites
Write-Host "1️⃣ Checking prerequisites..." -ForegroundColor Blue

# Check Firebase CLI
try {
    $firebaseVersion = firebase --version
    Write-Host "✅ Firebase CLI: $firebaseVersion" -ForegroundColor Green
} catch {
    Write-Host "❌ Firebase CLI not found. Installing..." -ForegroundColor Red
    npm install -g firebase-tools
}

# Check Flutter
try {
    $flutterVersion = flutter --version | Select-String "Flutter"
    Write-Host "✅ Flutter: $flutterVersion" -ForegroundColor Green
} catch {
    Write-Host "❌ Flutter not found. Please install Flutter first." -ForegroundColor Red
    exit 1
}

# Navigate to mobile app
Set-Location "apps\sns\mobile"

Write-Host ""
Write-Host "2️⃣ Setting up Flutter web..." -ForegroundColor Blue
flutter config --enable-web

# Create web platform if not exists
if (!(Test-Path "web")) {
    flutter create --platforms=web .
    Write-Host "✅ Web platform added" -ForegroundColor Green
}

Write-Host ""
Write-Host "3️⃣ Installing dependencies..." -ForegroundColor Blue
flutter pub get

Write-Host ""
Write-Host "4️⃣ Building for production..." -ForegroundColor Blue
flutter build web --release --web-renderer html --base-href "/"

if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Build failed!" -ForegroundColor Red
    exit 1
}

Write-Host "✅ Build completed successfully" -ForegroundColor Green

Write-Host ""
Write-Host "5️⃣ Firebase authentication..." -ForegroundColor Blue
try {
    # Check if already logged in
    $currentProject = firebase use
    if ($currentProject -like "*Error*") {
        Write-Host "🔑 Logging into Firebase..." -ForegroundColor Blue
        firebase login
    }
} catch {
    Write-Host "🔑 Logging into Firebase..." -ForegroundColor Blue
    firebase login
}

Write-Host ""
Write-Host "6️⃣ Initializing Firebase project..." -ForegroundColor Blue

# Create firebase.json if it doesn't exist
if (!(Test-Path "firebase.json")) {
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
                    source = "**/*.@(js|css|map|json|woff2|woff|ttf|eot|svg|png|jpg|jpeg|gif|ico)"
                    headers = @(
                        @{
                            key = "Cache-Control"
                            value = "max-age=31536000"
                        }
                    )
                }
            )
        }
    } | ConvertTo-Json -Depth 10
    
    $firebaseConfig | Out-File -FilePath "firebase.json" -Encoding UTF8
    Write-Host "✅ firebase.json created" -ForegroundColor Green
}

# Create .firebaserc if it doesn't exist
if (!(Test-Path ".firebaserc")) {
    $firebaserc = @{
        projects = @{
            default = "narratives-development-26c2d"
        }
    } | ConvertTo-Json -Depth 10
    
    $firebaserc | Out-File -FilePath ".firebaserc" -Encoding UTF8
    Write-Host "✅ .firebaserc created" -ForegroundColor Green
}

Write-Host ""
Write-Host "7️⃣ Deploying to Firebase Hosting..." -ForegroundColor Blue

try {
    firebase deploy --only hosting --project narratives-development-26c2d
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host ""
        Write-Host "🎉 Deployment successful!" -ForegroundColor Green
        Write-Host ""
        Write-Host "🌐 Your app is live at:" -ForegroundColor Cyan
        Write-Host "   https://narratives-development-26c2d.web.app" -ForegroundColor White
        Write-Host "   https://narratives-development-26c2d.firebaseapp.com" -ForegroundColor White
        Write-Host ""
        Write-Host "📱 App Features Deployed:" -ForegroundColor Yellow
        Write-Host "   ✅ SNS Feed with demo posts" -ForegroundColor Green
        Write-Host "   ✅ User profile with Firestore integration" -ForegroundColor Green
        Write-Host "   ✅ Responsive web design" -ForegroundColor Green
        Write-Host "   ✅ Progressive Web App (PWA)" -ForegroundColor Green
        Write-Host "   ✅ Multi-tab navigation" -ForegroundColor Green
    } else {
        throw "Deployment failed with exit code $LASTEXITCODE"
    }
} catch {
    Write-Host "❌ Deployment failed: $_" -ForegroundColor Red
    Write-Host ""
    Write-Host "🔧 Troubleshooting:" -ForegroundColor Yellow
    Write-Host "   1. Check Firebase project permissions" -ForegroundColor White
    Write-Host "   2. Verify project ID: narratives-development-26c2d" -ForegroundColor White
    Write-Host "   3. Run: firebase login --reauth" -ForegroundColor White
    exit 1
}

Set-Location "..\..\.."

Write-Host ""
Write-Host "📊 Deployment Summary:" -ForegroundColor Cyan
Write-Host "   📦 Build size: $(if (Test-Path 'apps\sns\mobile\build\web') { (Get-ChildItem 'apps\sns\mobile\build\web' -Recurse | Measure-Object -Property Length -Sum).Sum / 1MB } else { 'Unknown' }) MB" -ForegroundColor White
Write-Host "   🌍 Global CDN: Enabled" -ForegroundColor White
Write-Host "   🔒 HTTPS: Enabled" -ForegroundColor White
Write-Host "   📱 PWA: Enabled" -ForegroundColor White
Write-Host ""
Write-Host "🎯 Next Steps:" -ForegroundColor Yellow
Write-Host "   • Test the live app: https://narratives-development-26c2d.web.app" -ForegroundColor White
Write-Host "   • Set up custom domain (optional)" -ForegroundColor White
Write-Host "   • Configure Firebase Analytics" -ForegroundColor White
Write-Host "   • Set up CI/CD with GitHub Actions" -ForegroundColor White
