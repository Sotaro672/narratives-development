# Helper script to run mobile app on different platforms

Write-Host "📱 Narratives SNS Mobile App Launcher" -ForegroundColor Green
Write-Host "====================================" -ForegroundColor Green

# Check if we're in the correct directory
$currentPath = Get-Location
if ($currentPath.Path -notlike "*narratives-development") {
    Write-Host "❌ Please run this script from the narratives-development root directory" -ForegroundColor Red
    exit 1
}

# Navigate to mobile app directory
if (Test-Path "apps\sns\mobile") {
    Set-Location "apps\sns\mobile"
} else {
    Write-Host "❌ Mobile app directory not found: apps\sns\mobile" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "🔍 Checking available devices..." -ForegroundColor Blue
flutter devices

Write-Host ""
Write-Host "📋 Available platforms:" -ForegroundColor Cyan
Write-Host "1. Web (Chrome)" -ForegroundColor White
Write-Host "2. Web (Edge)" -ForegroundColor White  
Write-Host "3. Windows Desktop" -ForegroundColor White
Write-Host "4. Android Emulator (if available)" -ForegroundColor White
Write-Host "5. Check for emulators" -ForegroundColor White
Write-Host "6. Exit" -ForegroundColor White

Write-Host ""
$choice = Read-Host "Select platform (1-6)"

switch ($choice) {
    "1" {
        Write-Host "🌐 Starting web app in Chrome..." -ForegroundColor Blue
        Write-Host "Setting up web support..." -ForegroundColor Blue
        flutter config --enable-web
        flutter create --platforms=web .
        flutter run -d chrome
    }
    "2" {
        Write-Host "🌐 Starting web app in Edge..." -ForegroundColor Blue
        Write-Host "Setting up web support..." -ForegroundColor Blue
        flutter config --enable-web
        flutter create --platforms=web .
        flutter run -d edge
    }
    "3" {
        Write-Host "💻 Starting Windows desktop app..." -ForegroundColor Blue
        Write-Host "Setting up Windows platform support..." -ForegroundColor Yellow
        flutter config --enable-windows-desktop
        flutter create --platforms=windows .
        flutter run -d windows
    }
    "4" {
        Write-Host "📱 Looking for Android devices..." -ForegroundColor Blue
        $devices = flutter devices 2>&1
        if ($devices -like "*android*") {
            flutter run
        } else {
            Write-Host "❌ No Android devices found. Please start an emulator first." -ForegroundColor Red
            Write-Host "Run option 5 to check for available emulators." -ForegroundColor Yellow
        }
    }
    "5" {
        Write-Host "🔍 Checking available emulators..." -ForegroundColor Blue
        $emulators = flutter emulators 2>&1
        Write-Host $emulators
        
        if ($emulators -like "*No emulators*" -or $emulators -like "*No emulator*") {
            Write-Host "❌ No emulators found. Please create one in Android Studio:" -ForegroundColor Red
            Write-Host "1. Open Android Studio" -ForegroundColor Yellow
            Write-Host "2. Tools > AVD Manager" -ForegroundColor Yellow
            Write-Host "3. Create Virtual Device" -ForegroundColor Yellow
            Write-Host "4. Follow the setup wizard" -ForegroundColor Yellow
        } else {
            Write-Host ""
            $emulatorId = Read-Host "Enter emulator ID to start (or press Enter to skip)"
            if ($emulatorId) {
                Write-Host "🚀 Starting emulator: $emulatorId" -ForegroundColor Blue
                flutter emulators --launch $emulatorId
                Start-Sleep -Seconds 15
                Write-Host "📱 Starting app on emulator..." -ForegroundColor Blue
                flutter run
            }
        }
    }
    "6" {
        Write-Host "👋 Exiting..." -ForegroundColor Blue
        Set-Location "..\..\.."
        exit
    }
    default {
        Write-Host "❌ Invalid choice. Please run the script again." -ForegroundColor Red
    }
}

Set-Location "..\..\.."
