# Flutter installation verification script

Write-Host "Verifying Flutter installation..." -ForegroundColor Green

# Check if Flutter is in PATH
try {
    $flutterVersion = flutter --version 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Flutter is properly installed:" -ForegroundColor Green
        flutter --version
    } else {
        throw "Flutter command failed"
    }
} catch {
    Write-Host "ERROR: Flutter is not found in PATH" -ForegroundColor Red
    Write-Host "Please ensure C:\flutter\bin is added to your PATH environment variable" -ForegroundColor Yellow
    exit 1
}

Write-Host ""
Write-Host "Running Flutter doctor..." -ForegroundColor Blue
flutter doctor

Write-Host ""
Write-Host "Setup project dependencies..." -ForegroundColor Blue
Set-Location "apps\sns\mobile"

if (Test-Path "pubspec.yaml") {
    try {
        flutter pub get
        Write-Host "Dependencies installed successfully" -ForegroundColor Green
    } catch {
        Write-Host "ERROR: Failed to install dependencies: $_" -ForegroundColor Red
    }
} else {
    Write-Host "ERROR: pubspec.yaml not found. Are you in the correct directory?" -ForegroundColor Red
}

Write-Host ""
Write-Host "Available devices:" -ForegroundColor Blue
flutter devices

Write-Host ""
Write-Host "Flutter setup verification completed!" -ForegroundColor Green
Write-Host ""
Write-Host "To run the app:" -ForegroundColor Yellow
Write-Host "1. Start an Android emulator or connect a device" -ForegroundColor White
Write-Host "2. Run: flutter run" -ForegroundColor White
