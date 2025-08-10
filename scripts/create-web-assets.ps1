# Create web assets for Flutter deployment

Write-Host "🎨 Creating Web Assets for Flutter App" -ForegroundColor Green
Write-Host "======================================" -ForegroundColor Green

Set-Location "apps\sns\mobile\web"

Write-Host "1️⃣ Creating favicon..." -ForegroundColor Blue

# Create a simple SVG favicon and convert to PNG
$svgIcon = @'
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">
  <defs>
    <linearGradient id="grad" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" style="stop-color:#1976D2;stop-opacity:1" />
      <stop offset="100%" style="stop-color:#1E88E5;stop-opacity:1" />
    </linearGradient>
  </defs>
  <circle cx="50" cy="50" r="45" fill="url(#grad)" stroke="white" stroke-width="2"/>
  <text x="50" y="60" text-anchor="middle" fill="white" font-family="Arial, sans-serif" font-size="24" font-weight="bold">N</text>
</svg>
'@

$svgIcon | Out-File -FilePath "favicon.svg" -Encoding UTF8

Write-Host "2️⃣ Creating icons directory..." -ForegroundColor Blue
if (!(Test-Path "icons")) {
    New-Item -ItemType Directory -Path "icons"
}

# Create simple PNG icons using PowerShell (basic implementation)
Write-Host "3️⃣ Creating placeholder icons..." -ForegroundColor Blue

# For now, copy the Flutter default icon or create a simple text-based icon
# In a real scenario, you'd use proper image generation tools

$iconSvg192 = @'
<svg xmlns="http://www.w3.org/2000/svg" width="192" height="192" viewBox="0 0 192 192">
  <defs>
    <linearGradient id="grad" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" style="stop-color:#1976D2;stop-opacity:1" />
      <stop offset="100%" style="stop-color:#1E88E5;stop-opacity:1" />
    </linearGradient>
  </defs>
  <rect width="192" height="192" rx="24" fill="url(#grad)"/>
  <circle cx="96" cy="96" r="60" fill="rgba(255,255,255,0.2)" stroke="white" stroke-width="3"/>
  <text x="96" y="110" text-anchor="middle" fill="white" font-family="Arial, sans-serif" font-size="48" font-weight="bold">N</text>
</svg>
'@

$iconSvg512 = @'
<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512" viewBox="0 0 512 512">
  <defs>
    <linearGradient id="grad" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" style="stop-color:#1976D2;stop-opacity:1" />
      <stop offset="100%" style="stop-color:#1E88E5;stop-opacity:1" />
    </linearGradient>
  </defs>
  <rect width="512" height="512" rx="64" fill="url(#grad)"/>
  <circle cx="256" cy="256" r="160" fill="rgba(255,255,255,0.2)" stroke="white" stroke-width="8"/>
  <text x="256" y="290" text-anchor="middle" fill="white" font-family="Arial, sans-serif" font-size="128" font-weight="bold">N</text>
</svg>
'@

$iconSvg192 | Out-File -FilePath "icons\Icon-192.svg" -Encoding UTF8
$iconSvg512 | Out-File -FilePath "icons\Icon-512.svg" -Encoding UTF8

Write-Host "4️⃣ Creating manifest.json..." -ForegroundColor Blue

$manifest = @{
    name = "Narratives SNS"
    short_name = "Narratives"
    start_url = "/"
    display = "standalone"
    background_color = "#1976D2"
    theme_color = "#1976D2"
    description = "Social Network for Stories"
    orientation = "portrait-primary"
    prefer_related_applications = $false
    icons = @(
        @{
            src = "icons/Icon-192.svg"
            sizes = "192x192"
            type = "image/svg+xml"
            purpose = "any maskable"
        },
        @{
            src = "icons/Icon-512.svg"
            sizes = "512x512"
            type = "image/svg+xml"
            purpose = "any maskable"
        }
    )
} | ConvertTo-Json -Depth 10

$manifest | Out-File -FilePath "manifest.json" -Encoding UTF8

Write-Host "5️⃣ Updating index.html..." -ForegroundColor Blue

$indexHtml = @'
<!DOCTYPE html>
<html>
<head>
  <base href="$FLUTTER_BASE_HREF">
  
  <meta charset="UTF-8">
  <meta content="IE=Edge" http-equiv="X-UA-Compatible">
  <meta name="description" content="Narratives SNS - A decentralized social network for stories">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <meta name="keywords" content="SNS, social network, stories, narratives, blockchain, NFT">

  <!-- Progressive Web App -->
  <meta name="mobile-web-app-capable" content="yes">
  <meta name="apple-mobile-web-app-capable" content="yes">
  <meta name="apple-mobile-web-app-status-bar-style" content="black">
  <meta name="apple-mobile-web-app-title" content="Narratives SNS">
  <link rel="apple-touch-icon" href="icons/Icon-192.svg">

  <!-- Favicon -->
  <link rel="icon" type="image/svg+xml" href="favicon.svg"/>

  <title>Narratives SNS - Social Network for Stories</title>
  <link rel="manifest" href="manifest.json">

  <!-- Firebase SDK v9 (modular) -->
  <script type="module">
    import { initializeApp } from 'https://www.gstatic.com/firebasejs/9.22.0/firebase-app.js';
    import { getAuth } from 'https://www.gstatic.com/firebasejs/9.22.0/firebase-auth.js';
    import { getFirestore } from 'https://www.gstatic.com/firebasejs/9.22.0/firebase-firestore.js';
    import { getStorage } from 'https://www.gstatic.com/firebasejs/9.22.0/firebase-storage.js';
    
    const firebaseConfig = {
      apiKey: "AIzaSyC8qL9XQw5_-qQGGpXHBZJGBSgOzjGvhxA",
      authDomain: "narratives-development-26c2d.firebaseapp.com",
      projectId: "narratives-development-26c2d",
      storageBucket: "narratives-development-26c2d.appspot.com",
      messagingSenderId: "229613581466",
      appId: "1:229613581466:web:8f0f88901cc5cdec123456"
    };

    const app = initializeApp(firebaseConfig);
    window.firebase = { 
      app, 
      auth: getAuth(app), 
      firestore: getFirestore(app),
      storage: getStorage(app)
    };
    
    console.log('Firebase initialized for Narratives SNS deployment');
  </script>

  <!-- Loading styles -->
  <style>
    body {
      margin: 0;
      font-family: 'Roboto', sans-serif;
    }
    .loading {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      height: 100vh;
      background: linear-gradient(135deg, #1976D2 0%, #1E88E5 100%);
      color: white;
    }
    .loading-spinner {
      width: 60px;
      height: 60px;
      border: 4px solid rgba(255,255,255,0.3);
      border-radius: 50%;
      border-top-color: white;
      animation: spin 1s ease-in-out infinite;
      margin-bottom: 20px;
    }
    .loading-text {
      font-size: 24px;
      font-weight: 500;
      margin-bottom: 8px;
    }
    .loading-subtitle {
      font-size: 16px;
      opacity: 0.8;
    }
    @keyframes spin {
      to { transform: rotate(360deg); }
    }
  </style>
</head>
<body>
  <div id="loading" class="loading">
    <div class="loading-spinner"></div>
    <div class="loading-text">Narratives SNS</div>
    <div class="loading-subtitle">Social Network for Stories</div>
  </div>

  <script>
    // Service worker version fallback
    window.serviceWorkerVersion = null;
    
    window.addEventListener('load', function(ev) {
      _flutter.loader.loadEntrypoint({
        serviceWorker: {
          serviceWorkerVersion: serviceWorkerVersion,
        },
        onEntrypointLoaded: function(engineInitializer) {
          engineInitializer.initializeEngine().then(function(appRunner) {
            document.getElementById('loading').style.display = 'none';
            appRunner.runApp();
          });
        }
      });
    });
  </script>
  <script src="flutter.js" defer></script>
</body>
</html>
'@

$indexHtml | Out-File -FilePath "index.html" -Encoding UTF8

Set-Location "..\..\.."

Write-Host ""
Write-Host "✅ Web assets created successfully!" -ForegroundColor Green
Write-Host "📁 Created files:" -ForegroundColor Yellow
Write-Host "   • favicon.svg" -ForegroundColor White
Write-Host "   • icons/Icon-192.svg" -ForegroundColor White
Write-Host "   • icons/Icon-512.svg" -ForegroundColor White
Write-Host "   • manifest.json" -ForegroundColor White
Write-Host "   • index.html (updated)" -ForegroundColor White
