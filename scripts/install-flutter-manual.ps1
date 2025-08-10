# Flutter manual installation script for Windows

Write-Host "Flutter manual installation starting..." -ForegroundColor Green

# Check administrator privileges
if (-NOT ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Write-Host "ERROR: This script requires administrator privileges" -ForegroundColor Red
    Write-Host "Right-click PowerShell and select 'Run as Administrator'" -ForegroundColor Yellow
    exit 1
}

# Check if Flutter SDK already exists
$flutterDir = "C:\flutter"
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
$downloadPath = "$env:TEMP\flutter_windows.zip"

Write-Host "Downloading Flutter SDK..." -ForegroundColor Blue
Write-Host "URL: $flutterUrl" -ForegroundColor Gray

try {
    # Remove existing download if exists
    if (Test-Path $downloadPath) {
        Remove-Item $downloadPath -Force
    }
    
    Invoke-WebRequest -Uri $flutterUrl -OutFile $downloadPath -UseBasicParsing
    Write-Host "Download completed successfully" -ForegroundColor Green
} catch {
    Write-Host "ERROR: Download failed: $_" -ForegroundColor Red
    exit 1
}

# Extract ZIP file
Write-Host "Extracting Flutter SDK..." -ForegroundColor Blue
try {
    Expand-Archive -Path $downloadPath -DestinationPath "C:\" -Force
    Write-Host "Extraction completed successfully" -ForegroundColor Green
} catch {
    Write-Host "ERROR: Extraction failed: $_" -ForegroundColor Red
    exit 1
}

# Add to PATH environment variable
Write-Host "Setting PATH environment variable..." -ForegroundColor Blue
$currentPath = [Environment]::GetEnvironmentVariable("PATH", [System.EnvironmentVariableTarget]::Machine)
$flutterBinPath = "C:\flutter\bin"

if ($currentPath -notlike "*$flutterBinPath*") {
    try {
        $newPath = $currentPath + ";" + $flutterBinPath
        [Environment]::SetEnvironmentVariable("PATH", $newPath, [System.EnvironmentVariableTarget]::Machine)
        Write-Host "Flutter added to PATH environment variable" -ForegroundColor Green
    } catch {
        Write-Host "ERROR: Failed to update PATH: $_" -ForegroundColor Red
        Write-Host "Please manually add C:\flutter\bin to your PATH" -ForegroundColor Yellow
    }
} else {
    Write-Host "Flutter is already in PATH environment variable" -ForegroundColor Green
}

# Clean up temporary file
try {
    Remove-Item $downloadPath -Force -ErrorAction SilentlyContinue
} catch {
    # Ignore cleanup errors
}

Write-Host ""
Write-Host "Flutter installation completed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "1. Start a new PowerShell session" -ForegroundColor White
Write-Host "2. Run 'flutter doctor' to verify installation" -ForegroundColor White
Write-Host "3. Install Android Studio for Android development" -ForegroundColor White
Write-Host ""
Write-Host "Android Studio: https://developer.android.com/studio" -ForegroundColor Cyan
Write-Host "VS Code Flutter extension: https://marketplace.visualstudio.com/items?itemName=Dart-Code.flutter" -ForegroundColor Cyan
