# Deploy SNS Mobile App to Firebase Hosting

Write-Host "🚀 Deploying SNS Mobile App to Firebase" -ForegroundColor Green
Write-Host "=======================================" -ForegroundColor Green

# Check if Firebase CLI is installed
try {
    firebase --version | Out-Null
    Write-Host "✅ Firebase CLI is installed" -ForegroundColor Green
} catch {
    Write-Host "❌ Firebase CLI is not installed" -ForegroundColor Red
    Write-Host "Please install Firebase CLI:" -ForegroundColor Yellow
    Write-Host "npm install -g firebase-tools" -ForegroundColor Cyan
    exit 1
}

# Navigate to mobile app directory
Set-Location "apps\sns\mobile"

Write-Host ""
Write-Host "1️⃣ Setting up web support..." -ForegroundColor Blue
flutter config --enable-web

# Create web directory if it doesn't exist
if (!(Test-Path "web")) {
    flutter create --platforms=web .
}

Write-Host ""
Write-Host "2️⃣ Installing dependencies..." -ForegroundColor Blue
flutter pub get

Write-Host ""
Write-Host "3️⃣ Building web app for production..." -ForegroundColor Blue
flutter build web --release --web-renderer html

Write-Host ""
Write-Host "4️⃣ Firebase login check..." -ForegroundColor Blue
try {
    $loginStatus = firebase projects:list 2>&1
    if ($loginStatus -like "*Error*" -or $loginStatus -like "*not logged in*") {
        Write-Host "🔑 Logging into Firebase..." -ForegroundColor Blue
        firebase login
    } else {
        Write-Host "✅ Already logged into Firebase" -ForegroundColor Green
    }
} catch {
    Write-Host "🔑 Logging into Firebase..." -ForegroundColor Blue
    firebase login
}

Write-Host ""
Write-Host "5️⃣ Initializing Firebase project..." -ForegroundColor Blue
if (!(Test-Path "firebase.json")) {
    firebase init hosting --project narratives-development-26c2d
} else {
    Write-Host "✅ Firebase already initialized" -ForegroundColor Green
}

Write-Host ""
Write-Host "6️⃣ Deploying to Firebase Hosting..." -ForegroundColor Blue
firebase deploy --only hosting --project narratives-development-26c2d

Write-Host ""
Write-Host "🎉 Deployment completed!" -ForegroundColor Green
Write-Host ""
Write-Host "Your app is now available at:" -ForegroundColor Cyan
Write-Host "https://narratives-development-26c2d.web.app" -ForegroundColor Cyan
Write-Host "https://narratives-development-26c2d.firebaseapp.com" -ForegroundColor Cyan

Set-Location "..\..\.."
