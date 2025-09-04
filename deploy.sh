#!/bin/bash

# D2MCP Google Cloud Run Deployment Script
# This script builds and deploys the d2mcp application to Google Cloud Run

set -e

# Configuration
PROJECT_ID="ai-diagram-maker-471119"
SERVICE_NAME="ai-diagram-maker-d2mcp"
REGION="us-central1"
IMAGE_NAME="gcr.io/${PROJECT_ID}/${SERVICE_NAME}"
SERVICE_ACCOUNT=""
MEMORY="512Mi"
CPU="1"
MAX_INSTANCES="10"
MIN_INSTANCES="0"
CONCURRENCY="100"
TIMEOUT="300"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if required tools are installed
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    if ! command -v gcloud &> /dev/null; then
        print_error "gcloud CLI is not installed. Please install it from https://cloud.google.com/sdk/docs/install"
        exit 1
    fi
    
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed. Please install it from https://docs.docker.com/get-docker/"
        exit 1
    fi
    
    print_success "All prerequisites are installed"
}

# Function to validate configuration
validate_config() {
    if [ -z "$PROJECT_ID" ]; then
        print_error "PROJECT_ID is not set. Please set it in the script or export it as an environment variable."
        print_status "You can set it by running: export PROJECT_ID=your-project-id"
        exit 1
    fi
    
    print_success "Configuration validated"
}

# Function to authenticate with Google Cloud
authenticate() {
    print_status "Authenticating with Google Cloud..."
    
    if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" | grep -q .; then
        print_status "No active authentication found. Running gcloud auth login..."
        gcloud auth login
    fi
    
    print_status "Setting project to $PROJECT_ID..."
    gcloud config set project $PROJECT_ID
    
    print_status "Enabling required APIs..."
    gcloud services enable cloudbuild.googleapis.com
    gcloud services enable run.googleapis.com
    gcloud services enable containerregistry.googleapis.com
    
    print_success "Authentication completed"
}

# Function to build and push Docker image
build_and_push() {
    print_status "Building Docker image..."
    
    # Build the image
    docker build -t $IMAGE_NAME .
    
    print_status "Configuring Docker to use gcloud as a credential helper..."
    gcloud auth configure-docker
    
    print_status "Pushing image to Google Container Registry..."
    docker push $IMAGE_NAME
    
    print_success "Image built and pushed successfully"
}

# Function to deploy to Cloud Run
deploy() {
    print_status "Deploying to Google Cloud Run..."
    
    # Prepare deployment command
    DEPLOY_CMD="gcloud run deploy $SERVICE_NAME \
        --image $IMAGE_NAME \
        --platform managed \
        --region $REGION \
        --memory $MEMORY \
        --cpu $CPU \
        --max-instances $MAX_INSTANCES \
        --min-instances $MIN_INSTANCES \
        --concurrency $CONCURRENCY \
        --timeout $TIMEOUT \
        --allow-unauthenticated \
        --port 8080"
    
    # Add service account if specified
    if [ ! -z "$SERVICE_ACCOUNT" ]; then
        DEPLOY_CMD="$DEPLOY_CMD --service-account $SERVICE_ACCOUNT"
    fi
    
    # Execute deployment
    eval $DEPLOY_CMD
    
    print_success "Deployment completed successfully"
}

# Function to get service URL
get_service_url() {
    print_status "Getting service URL..."
    SERVICE_URL=$(gcloud run services describe $SERVICE_NAME --platform managed --region $REGION --format="value(status.url)")
    print_success "Service is available at: $SERVICE_URL"
    print_status "MCP Streamable HTTP endpoint: $SERVICE_URL/mcp"
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -p, --project-id PROJECT_ID    Google Cloud Project ID (required)"
    echo "  -s, --service-name NAME        Cloud Run service name (default: d2mcp)"
    echo "  -r, --region REGION           Cloud Run region (default: us-central1)"
    echo "  -m, --memory MEMORY           Memory allocation (default: 512Mi)"
    echo "  -c, --cpu CPU                 CPU allocation (default: 1)"
    echo "  --max-instances MAX           Maximum instances (default: 10)"
    echo "  --min-instances MIN           Minimum instances (default: 0)"
    echo "  --concurrency CONC            Concurrency limit (default: 100)"
    echo "  --timeout TIMEOUT             Request timeout in seconds (default: 300)"
    echo "  --service-account SA          Service account email"
    echo "  --build-only                  Only build and push image, don't deploy"
    echo "  --deploy-only                 Only deploy (assumes image already exists)"
    echo "  -h, --help                    Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  PROJECT_ID                    Google Cloud Project ID"
    echo "  SERVICE_NAME                  Cloud Run service name"
    echo "  REGION                        Cloud Run region"
    echo ""
    echo "Examples:"
    echo "  $0 -p my-project-id"
    echo "  $0 --project-id my-project-id --region europe-west1 --memory 1Gi"
    echo "  PROJECT_ID=my-project-id $0"
}

# Parse command line arguments
BUILD_ONLY=false
DEPLOY_ONLY=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--project-id)
            PROJECT_ID="$2"
            shift 2
            ;;
        -s|--service-name)
            SERVICE_NAME="$2"
            shift 2
            ;;
        -r|--region)
            REGION="$2"
            shift 2
            ;;
        -m|--memory)
            MEMORY="$2"
            shift 2
            ;;
        -c|--cpu)
            CPU="$2"
            shift 2
            ;;
        --max-instances)
            MAX_INSTANCES="$2"
            shift 2
            ;;
        --min-instances)
            MIN_INSTANCES="$2"
            shift 2
            ;;
        --concurrency)
            CONCURRENCY="$2"
            shift 2
            ;;
        --timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        --service-account)
            SERVICE_ACCOUNT="$2"
            shift 2
            ;;
        --build-only)
            BUILD_ONLY=true
            shift
            ;;
        --deploy-only)
            DEPLOY_ONLY=true
            shift
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Update IMAGE_NAME with current PROJECT_ID
IMAGE_NAME="gcr.io/${PROJECT_ID}/${SERVICE_NAME}"

# Main execution
main() {
    print_status "Starting D2MCP deployment to Google Cloud Run"
    print_status "Project ID: $PROJECT_ID"
    print_status "Service Name: $SERVICE_NAME"
    print_status "Region: $REGION"
    print_status "Image: $IMAGE_NAME"
    
    check_prerequisites
    validate_config
    authenticate
    
    if [ "$DEPLOY_ONLY" = false ]; then
        build_and_push
    fi
    
    if [ "$BUILD_ONLY" = false ]; then
        deploy
        get_service_url
    fi
    
    print_success "Deployment process completed!"
}

# Run main function
main
