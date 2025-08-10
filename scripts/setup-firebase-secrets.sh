#!/bin/bash

# Firebase secrets setup script for Narratives Development
set -e

echo "🔐 Setting up Firebase secrets..."

# Create secure directories
mkdir -p infrastructure/terraform/secrets
mkdir -p .secrets

# Move the Firebase key to secure location
SOURCE_FILE="narratives-development-26c2d-firebase-adminsdk-fbsvc-d82a4edbf6.json"
TARGET_FILE="infrastructure/terraform/secrets/firebase-admin-key.json"

if [ -f "$SOURCE_FILE" ]; then
    echo "📁 Moving Firebase key to secure location..."
    mv "$SOURCE_FILE" "$TARGET_FILE"
    chmod 600 "$TARGET_FILE"
    echo "✅ Firebase key moved to: $TARGET_FILE"
else
    echo "❌ Firebase key file not found: $SOURCE_FILE"
    exit 1
fi

# Set up environment variables file
ENV_FILE=".secrets/.env.firebase"
cat > "$ENV_FILE" << EOF
# Firebase Configuration
FIREBASE_PROJECT_ID=narratives-development-26c2d
FIREBASE_CLIENT_EMAIL=firebase-adminsdk-fbsvc@narratives-development-26c2d.iam.gserviceaccount.com
FIREBASE_PRIVATE_KEY_ID=d82a4edbf6a761df66a62887937a56b6d0f4e07a
GOOGLE_APPLICATION_CREDENTIALS=$PWD/$TARGET_FILE

# Firebase Web Config (get from Firebase Console)
NEXT_PUBLIC_FIREBASE_API_KEY=your-api-key-here
NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN=narratives-development-26c2d.firebaseapp.com
NEXT_PUBLIC_FIREBASE_PROJECT_ID=narratives-development-26c2d
NEXT_PUBLIC_FIREBASE_STORAGE_BUCKET=narratives-development-26c2d.appspot.com
NEXT_PUBLIC_FIREBASE_MESSAGING_SENDER_ID=your-sender-id-here
NEXT_PUBLIC_FIREBASE_APP_ID=your-app-id-here
EOF

chmod 600 "$ENV_FILE"
echo "✅ Environment file created: $ENV_FILE"

# Create Google Cloud Secret
if command -v gcloud >/dev/null 2>&1; then
    echo "☁️  Creating Google Cloud Secret Manager entry..."
    gcloud secrets create firebase-admin-service-account \
        --data-file="$TARGET_FILE" \
        --replication-policy=automatic 2>/dev/null || echo "Secret already exists"
    echo "✅ Google Cloud secret created"
fi

# Create Kubernetes secret
if command -v kubectl >/dev/null 2>&1 && kubectl cluster-info >/dev/null 2>&1; then
    echo "🚀 Creating Kubernetes secret..."
    kubectl create namespace narratives-dev --dry-run=client -o yaml | kubectl apply -f -
    kubectl create secret generic firebase-secrets \
        --from-file=admin-key="$TARGET_FILE" \
        --namespace=narratives-dev \
        --dry-run=client -o yaml | kubectl apply -f -
    echo "✅ Kubernetes secret created"
fi

echo ""
echo "🎉 Firebase secrets setup complete!"
echo ""
echo "Next steps:"
echo "1. Update $ENV_FILE with your Firebase web configuration"
echo "2. Source the environment file: source $ENV_FILE"
echo "3. Update terraform.tfvars with Firebase configuration"
echo ""
echo "⚠️  Important: Never commit the following files to Git:"
echo "   - $TARGET_FILE"
echo "   - $ENV_FILE"
echo "   - terraform.tfvars"
