# Flutter セットアップスクリプト for Windows

Write-Host "🚀 Flutter セットアップを開始します..." -ForegroundColor Green

# Flutter SDKの確認
$flutterPath = Get-Command flutter -ErrorAction SilentlyContinue
if ($flutterPath) {
    Write-Host "✅ Flutter は既にインストールされています" -ForegroundColor Green
    flutter --version
} else {
    Write-Host "❌ Flutter がインストールされていません" -ForegroundColor Red
    Write-Host "📥 Flutter をインストールしてください:" -ForegroundColor Yellow
    Write-Host "1. https://docs.flutter.dev/get-started/install/windows からFlutter SDKをダウンロード" -ForegroundColor White
    Write-Host "2. C:\flutter に解凍" -ForegroundColor White
    Write-Host "3. システム環境変数 PATH に C:\flutter\bin を追加" -ForegroundColor White
    Write-Host "4. 新しいPowerShellセッションを開始" -ForegroundColor White
    Write-Host ""
    Write-Host "または、以下のコマンドで自動インストールできます:" -ForegroundColor Yellow
    Write-Host "winget install --id=Google.Flutter" -ForegroundColor Cyan
    exit 1
}

# Android Studio / Visual Studio の確認
Write-Host "🔍 開発環境をチェックしています..." -ForegroundColor Blue
flutter doctor

# 依存関係のインストール
Write-Host "📦 Flutter の依存関係をインストールしています..." -ForegroundColor Blue
Set-Location "apps\sns\mobile"

try {
    flutter pub get
    Write-Host "✅ 依存関係のインストールが完了しました" -ForegroundColor Green
} catch {
    Write-Host "❌ 依存関係のインストールに失敗しました: $_" -ForegroundColor Red
    exit 1
}

# プラットフォーム固有のセットアップ
Write-Host "🛠️  プラットフォーム固有のセットアップを実行しています..." -ForegroundColor Blue

# Android セットアップ
if (Test-Path "android") {
    Write-Host "📱 Android プロジェクトをセットアップしています..." -ForegroundColor Blue
    try {
        flutter build apk --debug
        Write-Host "✅ Android デバッグビルドが成功しました" -ForegroundColor Green
    } catch {
        Write-Host "⚠️  Android ビルドに問題があります。flutter doctor で確認してください" -ForegroundColor Yellow
    }
}

# iOS セットアップ（Windowsでは利用不可）
Write-Host "ℹ️  iOS開発はmacOSでのみ利用可能です" -ForegroundColor Yellow

Write-Host ""
Write-Host "🎉 Flutter セットアップが完了しました！" -ForegroundColor Green
Write-Host ""
Write-Host "次のステップ:" -ForegroundColor Yellow
Write-Host "1. Android デバイス/エミュレータを起動" -ForegroundColor White
Write-Host "2. flutter run コマンドでアプリを実行" -ForegroundColor White
Write-Host "3. VS Code の Flutter 拡張機能をインストール" -ForegroundColor White
