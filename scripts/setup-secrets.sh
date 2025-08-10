#!/bin/bash

# Firebase secrets setup script
# This script helps you securely set up Firebase credentials

set -e

echo "🔐 Setting up Firebase secrets for Narratives Development"

# Check if required tools are installed
command -v gcloud >/dev/null 2>&1 || { echo "❌ gcloud CLI is required but not installed. Aborting." >&2; exit 1; }
command -v kubectl >/dev/null 2>&1 || { echo "❌ kubectl is required but not installed. Aborting." >&2; exit 1; }

# Set project
PROJECT_ID="narratives-development"
gcloud config set project $PROJECT_ID

echo "📁 Creating secrets directory..."
mkdir -p infrastructure/terraform/secrets

echo "🔑 Please place your Firebase admin service account key file at:"
echo "   infrastructure/terraform/secrets/firebase-admin-key.json"
echo ""
echo "📝 You can download this file from:"
echo "   Firebase Console > Project Settings > Service Accounts > Generate new private key"
echo ""

read -p "Press Enter after you've placed the firebase-admin-key.json file..."

# Verify the file exists
if [ ! -f "infrastructure/terraform/secrets/firebase-admin-key.json" ]; then
    echo "❌ Firebase admin key file not found!"
    exit 1
fi

echo "✅ Firebase admin key file found"

# Create Kubernetes secrets if cluster is available
if kubectl cluster-info >/dev/null 2>&1; then
    echo "🚀 Creating Kubernetes secrets..."
    
    # Create namespace if it doesn't exist
    kubectl create namespace narratives-dev --dry-run=client -o yaml | kubectl apply -f -
    
    # Create Firebase admin secret
    kubectl create secret generic firebase-secrets \
        --from-file=admin-key=infrastructure/terraform/secrets/firebase-admin-key.json \
        --namespace=narratives-dev \
        --dry-run=client -o yaml | kubectl apply -f -
    
    echo "✅ Kubernetes secrets created"
else
    echo "⚠️  Kubernetes cluster not available. Secrets will be created during deployment."
fi

# Create Google Cloud Secret Manager secrets
echo "☁️  Creating Google Cloud Secret Manager secrets..."

gcloud secrets create firebase-admin-service-account \
    --data-file=infrastructure/terraform/secrets/firebase-admin-key.json \
    --replication-policy=automatic || echo "Secret already exists"

echo "✅ Google Cloud Secret Manager secrets created"

echo ""
echo "🎉 Firebase secrets setup complete!"
echo ""
echo "Next steps:"
echo "1. Copy infrastructure/terraform/terraform.tfvars.example to terraform.tfvars"
echo "2. Fill in your Firebase configuration values in terraform.tfvars"
echo "3. Run 'terraform plan' and 'terraform apply' to deploy infrastructure"
