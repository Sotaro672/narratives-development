# Android 開発環境セットアップスクリプト

Write-Host "📱 Android 開発環境をセットアップします..." -ForegroundColor Green

# Android Studio の確認
$androidStudioPaths = @(
    "${env:ProgramFiles}\Android\Android Studio",
    "${env:ProgramFiles(x86)}\Android\Android Studio",
    "${env:LOCALAPPDATA}\Programs\Android Studio"
)

$androidStudioFound = $false
foreach ($path in $androidStudioPaths) {
    if (Test-Path $path) {
        Write-Host "✅ Android Studio が見つかりました: $path" -ForegroundColor Green
        $androidStudioFound = $true
        break
    }
}

if (-not $androidStudioFound) {
    Write-Host "❌ Android Studio が見つかりません" -ForegroundColor Red
    Write-Host "📥 Android Studio をダウンロードしてインストールしてください:" -ForegroundColor Yellow
    Write-Host "https://developer.android.com/studio" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "インストール後、以下の手順を実行してください:" -ForegroundColor Yellow
    Write-Host "1. Android Studio を起動" -ForegroundColor White
    Write-Host "2. SDK Manager を開く (Tools > SDK Manager)" -ForegroundColor White
    Write-Host "3. Android SDK Platform と Android SDK Build-Tools をインストール" -ForegroundColor White
    Write-Host "4. AVD Manager でエミュレータを作成" -ForegroundColor White
    return
}

# Android SDK の確認
$androidSdkPaths = @(
    "${env:LOCALAPPDATA}\Android\Sdk",
    "${env:APPDATA}\Android\Sdk",
    "${env:ANDROID_HOME}",
    "${env:ANDROID_SDK_ROOT}"
)

$androidSdkFound = $false
foreach ($path in $androidSdkPaths) {
    if ($path -and (Test-Path $path)) {
        Write-Host "✅ Android SDK が見つかりました: $path" -ForegroundColor Green
        $env:ANDROID_HOME = $path
        $env:ANDROID_SDK_ROOT = $path
        $androidSdkFound = $true
        break
    }
}

if (-not $androidSdkFound) {
    Write-Host "⚠️  Android SDK が見つかりません" -ForegroundColor Yellow
    Write-Host "Android Studio のSDK Managerから Android SDK をインストールしてください" -ForegroundColor Yellow
}

# Flutter doctor の実行
Write-Host "🔍 Flutter の環境を確認しています..." -ForegroundColor Blue
try {
    flutter doctor
    Write-Host ""
    Write-Host "🎯 Android 開発に必要な追加手順:" -ForegroundColor Yellow
    Write-Host "1. 'flutter doctor --android-licenses' を実行してライセンスに同意" -ForegroundColor White
    Write-Host "2. Android Studio でエミュレータを作成・起動" -ForegroundColor White
    Write-Host "3. 'flutter devices' でデバイスを確認" -ForegroundColor White
    Write-Host "4. 'flutter run' でアプリを実行" -ForegroundColor White
} catch {
    Write-Host "❌ Flutter コマンドが見つかりません" -ForegroundColor Red
    Write-Host "新しいPowerShellセッションを開始するか、Flutter を再インストールしてください" -ForegroundColor Yellow
}
