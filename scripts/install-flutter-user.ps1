# Flutter user installation script (no admin rights required)

Write-Host "Installing Flutter to user directory..." -ForegroundColor Green

# Set Flutter installation directory to user profile
$flutterDir = "$env:USERPROFILE\flutter"
$flutterBinPath = "$flutterDir\bin"

# Check if Flutter SDK already exists
if (Test-Path $flutterDir) {
    Write-Host "WARNING: Flutter is already installed at $flutterDir" -ForegroundColor Yellow
    $response = Read-Host "Do you want to reinstall? (y/N)"
    if ($response -ne "y" -and $response -ne "Y") {
        Write-Host "Installation cancelled" -ForegroundColor Yellow
        exit 0
    }
    try {
        Remove-Item $flutterDir -Recurse -Force -ErrorAction Stop
        Write-Host "Removed existing Flutter installation" -ForegroundColor Green
    } catch {
        Write-Host "ERROR: Failed to remove existing Flutter: $_" -ForegroundColor Red
        exit 1
    }
}

# Download Flutter SDK
$flutterUrl = "https://storage.googleapis.com/flutter_infra_release/releases/stable/windows/flutter_windows_3.16.0-stable.zip"
$downloadPath = "$env:TEMP\flutter_windows_user.zip"

Write-Host "Downloading Flutter SDK..." -ForegroundColor Blue
Write-Host "URL: $flutterUrl" -ForegroundColor Gray
Write-Host "Installing to: $flutterDir" -ForegroundColor Gray

try {
    # Remove existing download if exists
    if (Test-Path $downloadPath) {
        Remove-Item $downloadPath -Force
    }
    
    # Download with progress
    $webClient = New-Object System.Net.WebClient
    $webClient.DownloadFile($flutterUrl, $downloadPath)
    Write-Host "Download completed successfully" -ForegroundColor Green
} catch {
    Write-Host "ERROR: Download failed: $_" -ForegroundColor Red
    exit 1
}

# Extract ZIP file to user directory
Write-Host "Extracting Flutter SDK..." -ForegroundColor Blue
try {
    Expand-Archive -Path $downloadPath -DestinationPath $env:USERPROFILE -Force
    Write-Host "Extraction completed successfully" -ForegroundColor Green
} catch {
    Write-Host "ERROR: Extraction failed: $_" -ForegroundColor Red
    exit 1
}

# Add to user PATH environment variable
Write-Host "Setting user PATH environment variable..." -ForegroundColor Blue
try {
    $currentUserPath = [Environment]::GetEnvironmentVariable("PATH", [System.EnvironmentVariableTarget]::User)
    
    if ($currentUserPath -notlike "*$flutterBinPath*") {
        $newUserPath = if ($currentUserPath) { $currentUserPath + ";" + $flutterBinPath } else { $flutterBinPath }
        [Environment]::SetEnvironmentVariable("PATH", $newUserPath, [System.EnvironmentVariableTarget]::User)
        Write-Host "Flutter added to user PATH environment variable" -ForegroundColor Green
    } else {
        Write-Host "Flutter is already in user PATH environment variable" -ForegroundColor Green
    }
} catch {
    Write-Host "ERROR: Failed to update user PATH: $_" -ForegroundColor Red
    Write-Host "Please manually add $flutterBinPath to your user PATH" -ForegroundColor Yellow
}

# Update current session PATH
$env:PATH = $env:PATH + ";" + $flutterBinPath

# Clean up temporary file
try {
    Remove-Item $downloadPath -Force -ErrorAction SilentlyContinue
} catch {
    # Ignore cleanup errors
}

# Verify installation
Write-Host ""
Write-Host "Verifying Flutter installation..." -ForegroundColor Blue
try {
    & "$flutterBinPath\flutter.bat" --version
    Write-Host "Flutter installation verified successfully!" -ForegroundColor Green
} catch {
    Write-Host "WARNING: Flutter verification failed. You may need to restart your terminal." -ForegroundColor Yellow
}

Write-Host ""
Write-Host "Flutter installation completed successfully!" -ForegroundColor Green
Write-Host "Installation location: $flutterDir" -ForegroundColor Gray
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "1. Restart your PowerShell session or run: refreshenv" -ForegroundColor White
Write-Host "2. Run 'flutter doctor' to verify installation" -ForegroundColor White
Write-Host "3. Install Android Studio for Android development" -ForegroundColor White
Write-Host ""
Write-Host "Android Studio: https://developer.android.com/studio" -ForegroundColor Cyan
Write-Host "VS Code Flutter extension: https://marketplace.visualstudio.com/items?itemName=Dart-Code.flutter" -ForegroundColor Cyan
