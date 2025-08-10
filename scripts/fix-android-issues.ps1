# Fix Android development issues

Write-Host "Fixing Android development issues..." -ForegroundColor Green

# Accept Android licenses
Write-Host ""
Write-Host "=== Accepting Android Licenses ===" -ForegroundColor Cyan
Write-Host "This will accept all Android SDK licenses automatically" -ForegroundColor Yellow
Write-Host "Press Enter to continue or Ctrl+C to cancel..."
Read-Host

try {
    echo y | flutter doctor --android-licenses
    Write-Host "Android licenses accepted successfully" -ForegroundColor Green
} catch {
    Write-Host "Failed to accept licenses automatically. Try manual acceptance:" -ForegroundColor Red
    Write-Host "flutter doctor --android-licenses" -ForegroundColor Yellow
}

# Update Android embedding
Write-Host ""
Write-Host "=== Updating Android Embedding ===" -ForegroundColor Cyan
Set-Location "apps\sns\mobile"

# Update android/app/build.gradle
$buildGradlePath = "android\app\build.gradle"
if (Test-Path $buildGradlePath) {
    Write-Host "Updating build.gradle..." -ForegroundColor Blue
    
    $buildGradle = Get-Content $buildGradlePath -Raw
    
    # Update compile SDK version
    if ($buildGradle -notlike "*compileSdkVersion 34*") {
        $buildGradle = $buildGradle -replace 'compileSdkVersion flutter.compileSdkVersion', 'compileSdkVersion 34'
    }
    
    # Update target SDK version
    if ($buildGradle -notlike "*targetSdkVersion 34*") {
        $buildGradle = $buildGradle -replace 'targetSdkVersion flutter.targetSdkVersion', 'targetSdkVersion 34'
    }
    
    # Update min SDK version
    if ($buildGradle -notlike "*minSdkVersion 21*") {
        $buildGradle = $buildGradle -replace 'minSdkVersion flutter.minSdkVersion', 'minSdkVersion 21'
    }
    
    Set-Content $buildGradlePath $buildGradle
    Write-Host "build.gradle updated" -ForegroundColor Green
}

# Update AndroidManifest.xml
$manifestPath = "android\app\src\main\AndroidManifest.xml"
if (Test-Path $manifestPath) {
    Write-Host "Updating AndroidManifest.xml..." -ForegroundColor Blue
    
    $manifest = Get-Content $manifestPath -Raw
    
    # Update to V2 embedding
    $manifest = $manifest -replace 'android:name="io.flutter.app.FlutterActivity"', 'android:name="io.flutter.embedding.android.FlutterActivity"'
    
    # Add V2 embedding meta-data if not present
    if ($manifest -notlike "*flutterEmbedding*") {
        $manifest = $manifest -replace '</application>', @"
        <meta-data
            android:name="flutterEmbedding"
            android:value="2" />
    </application>
"@
    }
    
    Set-Content $manifestPath $manifest
    Write-Host "AndroidManifest.xml updated to V2 embedding" -ForegroundColor Green
}

# Clean and rebuild
Write-Host ""
Write-Host "Cleaning and rebuilding..." -ForegroundColor Blue
flutter clean
flutter pub get

Set-Location "..\..\.."

Write-Host ""
Write-Host "Android issues fixed!" -ForegroundColor Green
Write-Host ""
Write-Host "Verify with: flutter doctor" -ForegroundColor Yellow
Write-Host "Test with: cd apps\sns\mobile && flutter run" -ForegroundColor Yellow
