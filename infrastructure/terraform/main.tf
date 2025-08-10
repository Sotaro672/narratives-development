terraform {
  required_version = ">= 1.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

variable "project_id" {
  description = "GCP Project ID"
  type        = string
  default     = "narratives-development"
}

variable "region" {
  description = "GCP Region"
  type        = string
  default     = "asia-northeast1"
}

# GKE Cluster
resource "google_container_cluster" "narratives_cluster" {
  name     = "narratives-cluster"
  location = var.region

  initial_node_count = 1
  remove_default_node_pool = true

  network    = google_compute_network.vpc.name
  subnetwork = google_compute_subnetwork.subnet.name
}

# VPC
resource "google_compute_network" "vpc" {
  name                    = "narratives-vpc"
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "subnet" {
  name          = "narratives-subnet"
  ip_cidr_range = "10.10.0.0/24"
  region        = var.region
  network       = google_compute_network.vpc.id
}

# Cloud SQL
resource "google_sql_database_instance" "postgres" {
  name             = "narratives-postgres"
  database_version = "POSTGRES_15"
  region           = var.region

  settings {
    tier = "db-f1-micro"
  }

  deletion_protection = false
}

# Secret Manager for Firebase credentials
resource "google_secret_manager_secret" "firebase_admin_key" {
  secret_id = "firebase-admin-service-account"
  
  labels = {
    environment = "development"
    service     = "firebase"
  }

  replication {
    automatic = true
  }
}

resource "google_secret_manager_secret_version" "firebase_admin_key_version" {
  secret      = google_secret_manager_secret.firebase_admin_key.id
  secret_data = file("${path.module}/secrets/firebase-admin-key.json")
}

# Secret for Firebase web config
resource "google_secret_manager_secret" "firebase_web_config" {
  secret_id = "firebase-web-config"
  
  labels = {
    environment = "development"
    service     = "firebase"
  }

  replication {
    automatic = true
  }
}

resource "google_secret_manager_secret_version" "firebase_web_config_version" {
  secret      = google_secret_manager_secret.firebase_web_config.id
  secret_data = jsonencode({
    apiKey            = var.firebase_api_key
    authDomain        = var.firebase_auth_domain
    projectId         = var.firebase_project_id
    storageBucket     = var.firebase_storage_bucket
    messagingSenderId = var.firebase_messaging_sender_id
    appId             = var.firebase_app_id
  })
}

# Service Account for accessing secrets
resource "google_service_account" "narratives_app" {
  account_id   = "narratives-app"
  display_name = "Narratives Application Service Account"
  description  = "Service account for Narratives microservices"
}

# Grant access to secrets
resource "google_secret_manager_secret_iam_member" "firebase_admin_key_accessor" {
  secret_id = google_secret_manager_secret.firebase_admin_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.narratives_app.email}"
}

resource "google_secret_manager_secret_iam_member" "firebase_web_config_accessor" {
  secret_id = google_secret_manager_secret.firebase_web_config.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.narratives_app.email}"
}

# Additional variables for Firebase configuration
variable "firebase_api_key" {
  description = "Firebase API Key"
  type        = string
  sensitive   = true
}

variable "firebase_auth_domain" {
  description = "Firebase Auth Domain"
  type        = string
  default     = "narratives-development-26c2d.firebaseapp.com"
}

variable "firebase_project_id" {
  description = "Firebase Project ID"
  type        = string
  default     = "narratives-development-26c2d"
}

variable "firebase_storage_bucket" {
  description = "Firebase Storage Bucket"
  type        = string
  default     = "narratives-development-26c2d.appspot.com"
}

variable "firebase_messaging_sender_id" {
  description = "Firebase Messaging Sender ID"
  type        = string
  sensitive   = true
}

variable "firebase_app_id" {
  description = "Firebase App ID"
  type        = string
  sensitive   = true
}

# Enable Secret Manager API
resource "google_project_service" "secret_manager" {
  service = "secretmanager.googleapis.com"
}

# Enable IAM API
resource "google_project_service" "iam" {
  service = "iam.googleapis.com"
}
