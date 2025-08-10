# Flutter セットアップガイド（Windows）

## 1. Flutter SDK の手動インストール

### 方法 A: スクリプトを使用（推奨）
```powershell
# 管理者権限でPowerShellを実行
.\scripts\install-flutter-manual.ps1
```

### 方法 B: 手動でインストール

1. **Flutter SDK をダウンロード**
   - https://docs.flutter.dev/get-started/install/windows
   - `flutter_windows_3.16.0-stable.zip` をダウンロード

2. **解凍**
   ```
   C:\flutter\ に解凍
   ```

3. **PATH環境変数に追加**
   ```
   システム環境変数 PATH に C:\flutter\bin を追加
   ```

## 2. Android 開発環境のセットアップ

### Android Studio のインストール
1. https://developer.android.com/studio からダウンロード
2. インストール時に Android SDK も一緒にインストール

### Android SDK の設定
```powershell
# ライセンスに同意
flutter doctor --android-licenses

# 環境確認
flutter doctor
```

## 3. プロジェクトのセットアップ

```powershell
# プロジェクトディレクトリに移動
cd apps\sns\mobile

# 依存関係をインストール
flutter pub get

# 利用可能なデバイスを確認
flutter devices

# アプリを実行
flutter run
```

## 4. トラブルシューティング

### よくある問題と解決方法

**問題**: `flutter` コマンドが認識されない
```powershell
# 解決方法
# 1. 新しいPowerShellセッションを開始
# 2. PATH環境変数を確認
echo $env:PATH
# 3. Flutterのパスが含まれているか確認
```

**問題**: Android licenses が未同意
```powershell
# 解決方法
flutter doctor --android-licenses
# すべてのライセンスに 'y' で同意
```

**問題**: Android SDK が見つからない
```powershell
# 解決方法
# Android Studio の SDK Manager から Android SDK をインストール
# 環境変数 ANDROID_HOME を設定
```

## 5. 開発のヒント

### VS Code セットアップ
1. Flutter拡張機能をインストール
2. Dart拡張機能をインストール
3. `Ctrl + Shift + P` → `Flutter: New Project` でプロジェクト作成

### Android エミュレータ
```powershell
# エミュレータ一覧
flutter emulators

# エミュレータ起動
flutter emulators --launch <emulator_id>

# アプリ実行
flutter run
```

### ホットリロード
- ファイル保存時に自動反映
- `r` キーでホットリロード
- `R` キーでホットリスタート
