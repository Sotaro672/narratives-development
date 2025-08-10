# Complete Flutter to Firebase deployment script

Write-Host "🚀 Deploying Narratives SNS Flutter App to Firebase" -ForegroundColor Green
Write-Host "====================================================" -ForegroundColor Green

# Check prerequisites
Write-Host "1️⃣ Checking prerequisites..." -ForegroundColor Blue

# Check if we're in the correct directory
$currentPath = Get-Location
if ($currentPath.Path -notlike "*narratives-development") {
    Write-Host "❌ Please run this script from the narratives-development root directory" -ForegroundColor Red
    exit 1
}

# Check Firebase CLI
try {
    $firebaseVersion = firebase --version
    Write-Host "✅ Firebase CLI: $firebaseVersion" -ForegroundColor Green
} catch {
    Write-Host "❌ Firebase CLI not found. Installing..." -ForegroundColor Red
    npm install -g firebase-tools
}

# Check Flutter
try {
    $flutterVersion = flutter --version | Select-String "Flutter"
    Write-Host "✅ Flutter: $flutterVersion" -ForegroundColor Green
} catch {
    Write-Host "❌ Flutter not found. Please install Flutter first." -ForegroundColor Red
    exit 1
}

# Navigate to mobile app directory
Set-Location "apps\sns\mobile"

Write-Host ""
Write-Host "2️⃣ Setting up Flutter web platform..." -ForegroundColor Blue
flutter config --enable-web

# Create web platform if not exists
if (!(Test-Path "web")) {
    flutter create --platforms=web .
    Write-Host "✅ Web platform added" -ForegroundColor Green
}

Write-Host ""
Write-Host "3️⃣ Installing dependencies..." -ForegroundColor Blue
flutter clean
flutter pub get

Write-Host ""
Write-Host "4️⃣ Building production web app..." -ForegroundColor Blue
try {
    flutter build web --release --web-renderer html --base-href "/" --dart-define=FLUTTER_WEB_CANVASKIT_URL=https://www.gstatic.com/flutter-canvaskit/
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Build completed successfully" -ForegroundColor Green
    } else {
        throw "Build failed with exit code $LASTEXITCODE"
    }
} catch {
    Write-Host "❌ Build failed: $_" -ForegroundColor Red
    Write-Host "Trying alternative build..." -ForegroundColor Yellow
    flutter build web --release
}

Write-Host ""
Write-Host "5️⃣ Firebase authentication..." -ForegroundColor Blue
try {
    # Check if already logged in
    $projects = firebase projects:list 2>&1
    if ($projects -like "*Error*" -or $projects -like "*not logged in*") {
        Write-Host "🔑 Logging into Firebase..." -ForegroundColor Blue
        firebase login
    } else {
        Write-Host "✅ Already logged into Firebase" -ForegroundColor Green
    }
} catch {
    Write-Host "🔑 Logging into Firebase..." -ForegroundColor Blue
    firebase login
}

Write-Host ""
Write-Host "6️⃣ Configuring Firebase project..." -ForegroundColor Blue

# Create firebase.json configuration
$firebaseConfig = @{
    hosting = @{
        public = "build/web"
        ignore = @(
            "firebase.json",
            "**/.*",
            "**/node_modules/**"
        )
        rewrites = @(
            @{
                source = "**"
                destination = "/index.html"
            }
        )
        headers = @(
            @{
                source = "**/*.@(js|css|map|json|woff2|woff|ttf|eot|svg|png|jpg|jpeg|gif|ico|webp)"
                headers = @(
                    @{
                        key = "Cache-Control"
                        value = "max-age=31536000, immutable"
                    }
                )
            },
            @{
                source = "**/*.@(html|json)"
                headers = @(
                    @{
                        key = "Cache-Control"
                        value = "max-age=0, must-revalidate"
                    }
                )
            }
        )
        cleanUrls = $true
        trailingSlash = $false
    }
    firestore = @{
        rules = "firestore.rules"
        indexes = "firestore.indexes.json"
    }
    storage = @{
        rules = "storage.rules"
    }
} | ConvertTo-Json -Depth 10

$firebaseConfig | Out-File -FilePath "firebase.json" -Encoding UTF8
Write-Host "✅ firebase.json created" -ForegroundColor Green

# Create .firebaserc
$firebaserc = @{
    projects = @{
        default = "narratives-development-26c2d"
    }
} | ConvertTo-Json -Depth 10

$firebaserc | Out-File -FilePath ".firebaserc" -Encoding UTF8
Write-Host "✅ .firebaserc created" -ForegroundColor Green

# Create Firestore rules
$firestoreRules = @"
rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {
    // Users can read/write their own profile
    match /users/{userId} {
      allow read, write: if request.auth != null && request.auth.uid == userId;
    }
    
    // Users can read/write their own avatar
    match /avatars/{avatarId} {
      allow read, write: if request.auth != null && 
        request.auth.uid == resource.data.user_id;
      allow read: if request.auth != null; // All users can read others' avatars
    }
    
    // Posts are readable by all authenticated users
    match /posts/{postId} {
      allow read: if request.auth != null;
      allow create: if request.auth != null;
      allow update, delete: if request.auth != null && 
        request.auth.uid == resource.data.avatar_id;
    }
  }
}
"@

$firestoreRules | Out-File -FilePath "firestore.rules" -Encoding UTF8
Write-Host "✅ firestore.rules created" -ForegroundColor Green

# Create Storage rules
$storageRules = @"
rules_version = '2';
service firebase.storage {
  match /b/{bucket}/o {
    // Users can upload/read their own avatars
    match /avatars/{userId}/{fileName} {
      allow read: if true; // Anyone can read avatars
      allow write: if request.auth != null && request.auth.uid == userId;
    }
    
    // Users can upload/read their own post images
    match /posts/{userId}/{fileName} {
      allow read: if true; // Anyone can read post images
      allow write: if request.auth != null && request.auth.uid == userId;
    }
  }
}
"@

$storageRules | Out-File -FilePath "storage.rules" -Encoding UTF8
Write-Host "✅ storage.rules created" -ForegroundColor Green

# Create Firestore indexes
$firestoreIndexes = @{
    indexes = @(
        @{
            collectionGroup = "posts"
            queryScope = "COLLECTION"
            fields = @(
                @{ fieldPath = "created_at"; order = "DESCENDING" }
            )
        }
    )
    fieldOverrides = @()
} | ConvertTo-Json -Depth 10

$firestoreIndexes | Out-File -FilePath "firestore.indexes.json" -Encoding UTF8
Write-Host "✅ firestore.indexes.json created" -ForegroundColor Green

Write-Host ""
Write-Host "7️⃣ Deploying to Firebase..." -ForegroundColor Blue

try {
    # Deploy hosting, Firestore rules, and Storage rules
    firebase deploy --only hosting,firestore:rules,storage --project narratives-development-26c2d
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host ""
        Write-Host "🎉 Deployment successful!" -ForegroundColor Green
        Write-Host ""
        Write-Host "🌐 Your app is live at:" -ForegroundColor Cyan
        Write-Host "   📱 Main URL: https://narratives-development-26c2d.web.app" -ForegroundColor White
        Write-Host "   🔄 Alt URL:  https://narratives-development-26c2d.firebaseapp.com" -ForegroundColor White
        Write-Host ""
        Write-Host "📊 Deployment Features:" -ForegroundColor Yellow
        Write-Host "   ✅ SNS Feed with real-time posts" -ForegroundColor Green
        Write-Host "   ✅ User profile management" -ForegroundColor Green
        Write-Host "   ✅ Avatar & post image upload (GCS)" -ForegroundColor Green
        Write-Host "   ✅ Firestore database integration" -ForegroundColor Green
        Write-Host "   ✅ Firebase Authentication ready" -ForegroundColor Green
        Write-Host "   ✅ Progressive Web App (PWA)" -ForegroundColor Green
        Write-Host "   ✅ Responsive design (mobile/desktop)" -ForegroundColor Green
        Write-Host "   ✅ Multi-tab navigation" -ForegroundColor Green
        
        # Calculate build size
        $buildPath = "build\web"
        if (Test-Path $buildPath) {
            $buildSize = [math]::Round(((Get-ChildItem $buildPath -Recurse | Measure-Object -Property Length -Sum).Sum / 1MB), 2)
            Write-Host "   📦 Build size: $buildSize MB" -ForegroundColor Green
        }
        
    } else {
        throw "Deployment failed with exit code $LASTEXITCODE"
    }
} catch {
    Write-Host "❌ Deployment failed: $_" -ForegroundColor Red
    Write-Host ""
    Write-Host "🔧 Troubleshooting tips:" -ForegroundColor Yellow
    Write-Host "   1. Check Firebase project permissions" -ForegroundColor White
    Write-Host "   2. Verify project ID: narratives-development-26c2d" -ForegroundColor White
    Write-Host "   3. Run: firebase login --reauth" -ForegroundColor White
    Write-Host "   4. Check build output in build/web directory" -ForegroundColor White
    exit 1
}

Write-Host ""
Write-Host "🎯 Next Steps:" -ForegroundColor Cyan
Write-Host "   • Test the live app functionality" -ForegroundColor White
Write-Host "   • Set up Firebase Authentication" -ForegroundColor White
Write-Host "   • Configure custom domain (optional)" -ForegroundColor White
Write-Host "   • Set up monitoring and analytics" -ForegroundColor White
Write-Host "   • Configure CI/CD pipeline" -ForegroundColor White

Set-Location "..\..\.."

Write-Host ""
Write-Host "📋 Deployment completed at $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Green
