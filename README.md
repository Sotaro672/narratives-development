# Narratives Development

🚀 **マルチプラットフォーム対応のSNS・CRM・プロダクトカタログ統合システム**

ブロックチェーン技術とNFTを活用した分散型ソーシャルネットワーク

## ✨ 主な機能

- 📱 **マルチプラットフォーム対応**: Android、iOS、Web
- 🔗 **ブロックチェーン連携**: Solana NFT統合
- 🏢 **CRM機能**: 企業顧客管理
- 📦 **プロダクトカタログ**: デジタル商品管理
- 🛒 **Shopify連携**: ECサイト統合
- 🔥 **Firebase**: 認証・ホスティング
- 🎨 **Flutter UI**: モダンなユーザーインターフェース

## 🛠️ 技術スタック

### バックエンド
- **Go** + Google Cloud Platform (GKE)
- **GraphQL** (gqlgen) + **PostgreSQL**
- **Firebase Authentication**
- **Solana** ブロックチェーン統合

### フロントエンド
- **Flutter** (Mobile: Android/iOS)
- **React + Next.js** (Web)
- **TypeScript** + **Apollo Client**

### インフラ
- **Docker** + **Kubernetes**
- **Terraform** (Infrastructure as Code)
- **Firebase Hosting**
- **Google Cloud Platform**

## Firebase セットアップ

### 1. Firebase Admin SDKキーの配置
```bash
# セットアップスクリプトを実行
chmod +x scripts/setup-firebase-secrets.sh
./scripts/setup-firebase-secrets.sh
```

### 2. Firebase Web設定の取得
Firebase Console から以下の情報を取得し、`.secrets/.env.firebase` を更新してください：

- API Key
- Auth Domain  
- Project ID
- Storage Bucket
- Messaging Sender ID
- App ID

### 3. 環境変数の読み込み
```bash
source .secrets/.env.firebase
```

### 4. セキュリティ注意事項
以下のファイルは **絶対にGitにコミットしないでください**：
- `infrastructure/terraform/secrets/firebase-admin-key.json`
- `.secrets/.env.firebase`
- `terraform.tfvars`

## Flutter モバイルアプリ セットアップ

### 前提条件
- Flutter SDK 3.0+
- Android Studio または Visual Studio Code
- Android SDK（Android開発の場合）
- Xcode（iOS開発の場合、macOSのみ）

### 1. Flutter のインストール

#### Windows
```powershell
# Flutter を winget でインストール
winget install --id=Google.Flutter

# または手動でインストール
# 1. https://docs.flutter.dev/get-started/install/windows からダウンロード
# 2. C:\flutter に解凍
# 3. PATH環境変数に C:\flutter\bin を追加
```

#### macOS
```bash
# Homebrew でインストール
brew install --cask flutter

# または手動でインストール
# https://docs.flutter.dev/get-started/install/macos
```

### 2. 開発環境の確認
```bash
flutter doctor
```

### 3. SNS モバイルアプリのセットアップ
```bash
# プロジェクトディレクトリに移動
cd apps/sns/mobile

# 依存関係をインストール
flutter pub get

# Android デバイス/エミュレータでアプリを実行
flutter run

# または、特定のデバイスを指定
flutter devices  # 利用可能なデバイス一覧
flutter run -d <device-id>
```

### 4. 自動セットアップスクリプト（Windows）
```powershell
.\scripts\setup-flutter.ps1
```

### 5. 開発のヒント
- **Hot Reload**: コード変更時に `r` を押すと即座に反映
- **Hot Restart**: `R` を押すとアプリを再起動
- **VS Code**: Flutter拡張機能をインストールして開発効率向上
- **Android Emulator**: Android Studio で仮想デバイスを作成

### トラブルシューティング
```bash
# Flutterの問題診断
flutter doctor -v

# キャッシュをクリア
flutter clean
flutter pub get

# Gradleの問題（Android）
cd android
./gradlew clean
```

## 🚀 クイックスタート

### 1. 環境セットアップ
```bash
git clone https://github.com/Sotaro672/narratives-development.git
cd narratives-development

# 自動セットアップスクリプト実行
.\scripts\quick-setup.ps1
```

### 2. モバイルアプリ実行
```bash
# Flutter環境でWebアプリを起動
.\scripts\run-mobile-app.ps1

# または直接実行
cd apps\sns\mobile
flutter run -d chrome
```

### 3. Firebase デプロイ
```bash
.\scripts\deploy-mobile-to-firebase.ps1
```

## 📱 ライブデモ

- **SNS Mobile App**: https://narratives-development-26c2d.web.app
- **CRM Web App**: https://narratives-development-26c2d.firebaseapp.com

## 🏗️ プロジェクト構成

```
narratives-development/
├── apps/
│   ├── sns/mobile/          # Flutter SNSアプリ
│   ├── sns-web/            # React SNS Web
│   └── crm-web/            # React CRM Web
├── services/               # Go マイクロサービス
│   ├── sns-backend/        # SNS API
│   ├── crm-backend/        # CRM API
│   ├── catalog-backend/    # 商品カタログ API
│   └── token-registry/     # Solana トークン管理
├── infrastructure/         # インフラ設定
│   ├── terraform/          # GCP設定
│   └── k8s/               # Kubernetes設定
└── scripts/               # 自動化スクリプト
```

## 🔧 開発環境

### 必要なツール
- **Flutter** 3.16+
- **Node.js** 18+
- **Go** 1.21+
- **Docker** & **Docker Compose**
- **Firebase CLI**

### 開発サーバー起動
```bash
# バックエンドサービス
docker-compose -f docker-compose.microservices.yml up

# フロントエンド
npm run dev              # 全Webアプリ
flutter run -d chrome    # モバイルアプリ（Web版）
```

## 📄 ライセンス

MIT License

## 🤝 コントリビューション

プルリクエストや Issue の作成を歓迎します！

---

**Built with ❤️ using Flutter, Go, and Blockchain Technology**
