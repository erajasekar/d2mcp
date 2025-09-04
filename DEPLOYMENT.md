# D2MCP Google Cloud Run Deployment Guide

This guide provides instructions for deploying the D2MCP (D2 Model Context Protocol) server to Google Cloud Run.

## Prerequisites

1. **Google Cloud Account**: You need a Google Cloud account with billing enabled
2. **Google Cloud SDK**: Install the [Google Cloud SDK](https://cloud.google.com/sdk/docs/install)
3. **Docker**: Install [Docker](https://docs.docker.com/get-docker/)
4. **Project Setup**: Create a Google Cloud project and note the Project ID

## Quick Start

### 1. Set Environment Variables

```bash
export PROJECT_ID="your-google-cloud-project-id"
export SERVICE_NAME="ai-diagram-maker-d2mcp"  # Optional, defaults to d2mcp
export REGION="us-central1"  # Optional, defaults to us-central1
```

### 2. Deploy Using the Script

```bash
# Make the script executable
chmod +x deploy.sh

# Deploy with default settings
./deploy.sh -p $PROJECT_ID

# Or deploy with custom settings
./deploy.sh -p $PROJECT_ID -r europe-west1 -m 1Gi --max-instances 20
```

### 3. Access Your Service

After deployment, the script will display the service URL. The MCP Streamable HTTP endpoint will be available at:
```
https://your-service-url/mcp
```

## Manual Deployment

### 1. Authenticate with Google Cloud

```bash
gcloud auth login
gcloud config set project $PROJECT_ID
gcloud auth configure-docker
```

### 2. Enable Required APIs

```bash
gcloud services enable cloudbuild.googleapis.com
gcloud services enable run.googleapis.com
gcloud services enable containerregistry.googleapis.com
```

### 3. Build and Push Docker Image

```bash
# Build the image
docker build -t gcr.io/$PROJECT_ID/d2mcp .

# Push to Container Registry
docker push gcr.io/$PROJECT_ID/d2mcp
```

### 4. Deploy to Cloud Run

```bash
gcloud run deploy d2mcp \
  --image gcr.io/$PROJECT_ID/d2mcp \
  --platform managed \
  --region us-central1 \
  --memory 512Mi \
  --cpu 1 \
  --max-instances 10 \
  --min-instances 0 \
  --concurrency 100 \
  --timeout 300 \
  --allow-unauthenticated \
  --port 8080
```

## Configuration Options

### Service Configuration

| Parameter | Default | Description |
|-----------|---------|-------------|
| `--memory` | 512Mi | Memory allocation for each instance |
| `--cpu` | 1 | CPU allocation for each instance |
| `--max-instances` | 10 | Maximum number of instances |
| `--min-instances` | 0 | Minimum number of instances |
| `--concurrency` | 100 | Maximum concurrent requests per instance |
| `--timeout` | 300 | Request timeout in seconds |
| `--region` | us-central1 | Cloud Run region |

### Application Configuration

The application runs in **Streamable HTTP mode** by default for Cloud Run deployment with the following settings:

- **Transport**: Streamable HTTP
- **Port**: 8080 (Cloud Run will override with PORT env var)
- **Mode**: Stateless
- **CORS**: Enabled with wildcard origins
- **Credentials**: Disabled

## Automated Deployment with Cloud Build

### 1. Connect Repository to Cloud Build

```bash
# Connect your GitHub repository
gcloud builds triggers create github \
  --repo-name=your-repo-name \
  --repo-owner=your-github-username \
  --branch-pattern="^main$" \
  --build-config=cloudbuild.yaml
```

### 2. Deploy via Cloud Build

```bash
# Trigger a build manually
gcloud builds submit --config cloudbuild.yaml .
```

## Monitoring and Logs

### View Logs

```bash
# View recent logs
gcloud run logs read d2mcp --region us-central1

# Follow logs in real-time
gcloud run logs tail d2mcp --region us-central1
```

### Monitor Performance

1. Go to [Google Cloud Console](https://console.cloud.google.com)
2. Navigate to Cloud Run
3. Select your service
4. View metrics, logs, and performance data

## Troubleshooting

### Common Issues

1. **Authentication Errors**
   ```bash
   gcloud auth login
   gcloud auth configure-docker
   ```

2. **Permission Denied**
   - Ensure your account has the necessary IAM roles:
     - Cloud Run Admin
     - Cloud Build Editor
     - Storage Admin

3. **Build Failures**
   - Check that all dependencies are properly specified in `go.mod`
   - Verify Docker build context includes all necessary files

4. **Service Not Starting**
   - Check logs: `gcloud run logs read d2mcp --region us-central1`
   - Verify the application is listening on the correct port (8080)

### Health Checks

The Dockerfile includes a health check that verifies the service is responding:

```bash
# Check service health
curl -f https://your-service-url/health || echo "Service is not healthy"
```

## Cost Optimization

### Resource Tuning

- **Memory**: Start with 512Mi and adjust based on usage
- **CPU**: Use 1 CPU for most workloads
- **Instances**: Set min-instances to 0 for cost savings
- **Concurrency**: Higher concurrency = fewer instances needed

### Monitoring Costs

1. Go to Cloud Console > Billing
2. Set up budget alerts
3. Monitor Cloud Run usage in the billing dashboard

## Security Considerations

1. **Authentication**: The service is deployed with `--allow-unauthenticated` for public access
2. **CORS**: Configured with wildcard origins - consider restricting for production
3. **Service Account**: Consider using a dedicated service account for production
4. **HTTPS**: Cloud Run automatically provides HTTPS

## Scaling

Cloud Run automatically scales based on traffic:

- **Scale to Zero**: When no traffic, instances scale to 0
- **Auto-scaling**: Scales up to max-instances based on demand
- **Cold Starts**: First request after scale-to-zero may have higher latency

## Environment Variables

You can set environment variables for the Cloud Run service:

```bash
gcloud run services update d2mcp \
  --region us-central1 \
  --set-env-vars="D2_LOG_LEVEL=INFO,ENVIRONMENT=production"
```

## Support

For issues related to:
- **D2MCP Application**: Check the application logs and GitHub issues
- **Google Cloud Run**: Consult [Cloud Run documentation](https://cloud.google.com/run/docs)
- **Docker**: Check [Docker documentation](https://docs.docker.com/)
