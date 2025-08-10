# Complete development environment setup script

param(
    [switch]$SkipFlutter,
    [switch]$SkipAndroid,
    [switch]$UserInstall
)

Write-Host "Setting up Narratives Development Environment..." -ForegroundColor Green

# Function to check if running as administrator
function Test-Administrator {
    return ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")
}

# Install Flutter if not skipped
if (-not $SkipFlutter) {
    Write-Host ""
    Write-Host "=== Flutter Installation ===" -ForegroundColor Cyan
    
    if ($UserInstall -or -not (Test-Administrator)) {
        Write-Host "Installing Flutter to user directory (no admin rights required)..." -ForegroundColor Blue
        & ".\scripts\install-flutter-user.ps1"
    } else {
        & ".\scripts\install-flutter-manual.ps1"
    }
}

# Setup project
Write-Host ""
Write-Host "=== Project Setup ===" -ForegroundColor Cyan

# Install Node.js dependencies
if (Test-Path "package.json") {
    Write-Host "Installing Node.js dependencies..." -ForegroundColor Blue
    try {
        npm install
        Write-Host "Node.js dependencies installed successfully" -ForegroundColor Green
    } catch {
        Write-Host "ERROR: Failed to install Node.js dependencies: $_" -ForegroundColor Red
    }
}

# Setup Flutter project
if (Test-Path "apps\sns\mobile\pubspec.yaml") {
    Write-Host "Setting up Flutter project..." -ForegroundColor Blue
    Set-Location "apps\sns\mobile"
    
    try {
        flutter pub get
        Write-Host "Flutter dependencies installed successfully" -ForegroundColor Green
    } catch {
        Write-Host "ERROR: Failed to install Flutter dependencies" -ForegroundColor Red
        Write-Host "Make sure Flutter is properly installed and in PATH" -ForegroundColor Yellow
    }
    
    Set-Location "..\..\.."
}

# Check Android development environment
if (-not $SkipAndroid) {
    Write-Host ""
    Write-Host "=== Android Development Check ===" -ForegroundColor Cyan
    & ".\scripts\setup-android-development.ps1"
}

Write-Host ""
Write-Host "=== Environment Verification ===" -ForegroundColor Cyan
& ".\scripts\verify-flutter-installation.ps1"

Write-Host ""
Write-Host "Development environment setup completed!" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "1. Install Android Studio if not already installed" -ForegroundColor White
Write-Host "2. Create and start an Android emulator" -ForegroundColor White
Write-Host "3. Run: cd apps\sns\mobile && flutter run" -ForegroundColor White
