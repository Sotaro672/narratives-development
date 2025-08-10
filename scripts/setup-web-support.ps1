# Setup web support for Flutter app

Write-Host "🌐 Setting up Web Support for Narratives SNS" -ForegroundColor Green
Write-Host "=============================================" -ForegroundColor Green

Set-Location "apps\sns\mobile"

Write-Host ""
Write-Host "1️⃣ Enabling web platform..." -ForegroundColor Blue
flutter config --enable-web

Write-Host ""
Write-Host "2️⃣ Adding web platform to project..." -ForegroundColor Blue
flutter create --platforms=web .

Write-Host ""
Write-Host "3️⃣ Cleaning and getting dependencies..." -ForegroundColor Blue
flutter clean
flutter pub get

Write-Host ""
Write-Host "4️⃣ Testing web build..." -ForegroundColor Blue
try {
    flutter build web
    Write-Host "✅ Web build successful!" -ForegroundColor Green
} catch {
    Write-Host "⚠️ Web build had some issues, but should still work" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "5️⃣ Starting web app..." -ForegroundColor Blue
flutter run -d chrome

Set-Location "..\..\.."
