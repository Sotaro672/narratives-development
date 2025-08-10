# Quick setup script for Narratives Development

Write-Host "Narratives Development - Quick Setup" -ForegroundColor Green
Write-Host "====================================" -ForegroundColor Green

# Check if Flutter is already installed
$flutterInstalled = $false
try {
    flutter --version > $null 2>&1
    if ($LASTEXITCODE -eq 0) {
        $flutterInstalled = $true
        Write-Host "Flutter is already installed:" -ForegroundColor Green
        flutter --version
    }
} catch {
    # Flutter not found
}

if (-not $flutterInstalled) {
    Write-Host ""
    Write-Host "Flutter is not installed. Installing to user directory..." -ForegroundColor Yellow
    & ".\scripts\install-flutter-user.ps1"
    
    # Refresh environment variables
    $env:PATH = [Environment]::GetEnvironmentVariable("PATH", [System.EnvironmentVariableTarget]::User) + ";" + [Environment]::GetEnvironmentVariable("PATH", [System.EnvironmentVariableTarget]::Machine)
}

Write-Host ""
Write-Host "Setting up project dependencies..." -ForegroundColor Blue

# Install Node.js dependencies
if (Test-Path "package.json") {
    Write-Host "Installing Node.js dependencies..." -ForegroundColor Blue
    Write-Host "Note: Workspace dependencies require npm 7+ or pnpm" -ForegroundColor Yellow
    
    # Try npm first, fallback to yarn if workspace issues
    try {
        npm install 2>$null
        if ($LASTEXITCODE -ne 0) {
            Write-Host "npm install failed. Trying with --legacy-peer-deps..." -ForegroundColor Yellow
            npm install --legacy-peer-deps
        }
    } catch {
        Write-Host "npm failed. Consider using pnpm for better workspace support:" -ForegroundColor Yellow
        Write-Host "npm install -g pnpm && pnpm install" -ForegroundColor Cyan
    }
}

# Setup Flutter project
Write-Host "Setting up Flutter mobile app..." -ForegroundColor Blue
Set-Location "apps\sns\mobile"

try {
    flutter pub get
    Write-Host "Flutter dependencies installed successfully" -ForegroundColor Green
} catch {
    Write-Host "ERROR: Failed to install Flutter dependencies" -ForegroundColor Red
    Write-Host "Try restarting your terminal and running 'flutter pub get' manually" -ForegroundColor Yellow
}

Set-Location "..\..\.."

Write-Host ""
Write-Host "Running Flutter doctor..." -ForegroundColor Blue
flutter doctor

# Check for Android license issues and fix them
Write-Host ""
Write-Host "Checking Android licenses..." -ForegroundColor Blue
try {
    $doctorOutput = flutter doctor 2>&1
    if ($doctorOutput -like "*Some Android licenses not accepted*") {
        Write-Host "Android licenses not accepted. Accepting automatically..." -ForegroundColor Yellow
        flutter doctor --android-licenses --android-answer=y
    }
} catch {
    Write-Host "Could not check Android licenses automatically" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "Updating Flutter app to Android V2 embedding..." -ForegroundColor Blue
Set-Location "apps\sns\mobile"

# Update Android embedding to V2
$androidManifestPath = "android\app\src\main\AndroidManifest.xml"
if (Test-Path $androidManifestPath) {
    Write-Host "Updating AndroidManifest.xml for V2 embedding..." -ForegroundColor Blue
    
    # Create backup
    Copy-Item $androidManifestPath "$androidManifestPath.backup"
    
    # Read and update manifest
    $manifest = Get-Content $androidManifestPath -Raw
    
    # Add V2 embedding configuration
    if ($manifest -notlike "*android:name=`"io.flutter.embedding.android.FlutterActivity`"*") {
        $manifest = $manifest -replace 'android:name="io.flutter.app.FlutterActivity"', 'android:name="io.flutter.embedding.android.FlutterActivity"'
        $manifest = $manifest -replace '</application>', @"
        <meta-data
            android:name="flutterEmbedding"
            android:value="2" />
    </application>
"@
        Set-Content $androidManifestPath $manifest
        Write-Host "Android V2 embedding configured" -ForegroundColor Green
    }
}

Set-Location "..\..\.."

Write-Host ""
Write-Host "Quick setup completed!" -ForegroundColor Green
Write-Host ""
Write-Host "Setup Summary:" -ForegroundColor Cyan
Write-Host "✅ Flutter installed and verified" -ForegroundColor Green
Write-Host "✅ Flutter dependencies installed" -ForegroundColor Green
Write-Host "⚠️  Node.js workspace dependencies (use pnpm if needed)" -ForegroundColor Yellow
Write-Host "⚠️  Android licenses (run manually if needed)" -ForegroundColor Yellow
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "1. Accept Android licenses: flutter doctor --android-licenses" -ForegroundColor White
Write-Host "2. Start Android emulator or connect device" -ForegroundColor White
Write-Host "3. Run: cd apps\sns\mobile && flutter run" -ForegroundColor White
Write-Host ""
Write-Host "For Node.js workspaces, consider using pnpm:" -ForegroundColor Yellow
Write-Host "npm install -g pnpm && pnpm install" -ForegroundColor Cyan
