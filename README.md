# HCP Terraform Checkmarx One Integration

A Go-based web service that integrates HashiCorp Cloud Platform (HCP) Terraform with Checkmarx One for Infrastructure as Code (IaC) security scanning. This service acts as a run task handler, processing Terraform configurations and submitting them to Checkmarx One for KICS-based IaC scanning.

## Features

- **HCP Terraform Integration**: Full compliance with HCP Terraform Run Task API
- **Multiple Stage Support**: Supports both `pre_apply` and `post_plan` run task stages
- **Checkmarx One IaC Scanning**: KICS-based Infrastructure as Code security analysis
- **Dynamic Configuration**: Parse Terraform variables for scan customization
- **Policy Enforcement**: Break builds on policy violations
- **Rich Reporting**: Detailed findings with severity levels and remediation guidance
- **Multi-Region Support**: Support for all Checkmarx One regions
- **Container Ready**: OpenShift/Kubernetes deployment with CRI-O support

## Quick Start

### 1. Local Development

```bash
# Clone repository
git clone https://github.com/agehring/hcp-checkmarx-listener.git
cd hcp-checkmarx-listener

# Install dependencies
go mod download

# Configure environment
cp config.example.yaml config.yaml
# Edit config.yaml with your settings

# Run application
go run ./cmd/main.go
```

### 2. Container Deployment

#### OpenShift/Kubernetes

```bash
# Build container
./build-container.sh --tag v1.0.0

# Deploy to OpenShift
oc apply -f k8s/all-in-one.yaml

# Configure secrets
oc create secret generic hcp-checkmarx-secrets \
  --from-literal=checkmarx_api_key=$(echo 'your-api-key' | base64) \
  --from-literal=hmac_secret=$(echo 'your-hmac-secret' | base64) \
  -n hcp-checkmarx
```

See [OpenShift Deployment Guide](docs/openshift-deployment.md) for detailed instructions.

#### Docker/Podman

```bash
# Build image
docker build -t hcp-checkmarx-listener .

# Run container
docker run -d \
  -p 8080:8080 \
  -e CHECKMARX_API_KEY=your-api-key \
  -e HMAC_SECRET=your-hmac-secret \
  hcp-checkmarx-listener
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8080` |
| `CHECKMARX_API_KEY` | Checkmarx One API key | Required |
| `CHECKMARX_BASE_URL` | Checkmarx One API URL | `https://ast.checkmarx.net` |
| `CHECKMARX_REGION` | Checkmarx One region | `US` |
| `HMAC_SECRET` | HMAC secret for validation | Required |

### Supported Regions

| Region | Code | Base URL |
|--------|------|----------|
| US | US | https://ast.checkmarx.net |
| US2 | US2 | https://us.ast.checkmarx.net |
| EU | EU | https://eu.ast.checkmarx.net |
| EU2 | EUS | https://eu-2.ast.checkmarx.net |
| Germany | DEU | https://deu.ast.checkmarx.net |
| ANZ | ANZ | https://anz.ast.checkmarx.net |
| India | IND | https://ind.ast.checkmarx.net |
| Singapore | SNG | https://sng.ast.checkmarx.net |

### Dynamic Terraform Configuration

The service automatically parses Terraform configuration files to extract Checkmarx One scan parameters, allowing users to customize scans without modifying the service configuration.

#### Supported Variables

Users can specify scan parameters in their `variables.tf` files:

```hcl
:q
:q
:q

```

#### How It Works

1. **Download**: Service downloads Terraform workspace from HCP Terraform
2. **Parse**: Scans all `variables.tf` files in the workspace for Checkmarx variables
3. **Extract**: Retrieves default values from variable declarations
4. **Apply**: Uses extracted values for Checkmarx One scan submission
5. **Fallback**: Uses service defaults if variables are not found

#### Variable Precedence

1. **Terraform variables.tf defaults** (highest priority)
2. **Service configuration defaults** (fallback)
3. **Built-in defaults** (lowest priority)

#### Examples

**Basic Configuration:**
```hcl
variable "checkmarx_project_name" {
  default = "web-application-iac"
}
```

**Environment-Specific:**
```hcl
variable "checkmarx_project_name" {
  description = "Checkmarx project for this environment"
  type        = string
  default     = "prod-infrastructure"
}

variable "checkmarx_group" {
  description = "Security team group"
  type        = string
  default     = "Platform-Security"
}

variable "checkmarx_branch" {
  description = "Feature branch for this deployment"
  type        = string
  default     = "feature/new-vpc-setup"
}
```

**Multi-Environment Workspace:**
```hcl
locals {
  environment = terraform.workspace
  
  checkmarx_config = {
    dev = {
      project_name = "myapp-dev-iac"
      group        = "Development"
      branch       = "develop"
    }
    prod = {
      project_name = "myapp-prod-iac"
      group        = "Production"
      branch       = "main"
    }
  }
}

variable "checkmarx_project_name" {
  default = local.checkmarx_config[local.environment].project_name
}

variable "checkmarx_group" {
  default = local.checkmarx_config[local.environment].group
}

variable "checkmarx_branch" {
  default = local.checkmarx_config[local.environment].branch
}
```

## HCP Terraform Setup

### 1. Create Run Task

In your HCP Terraform organization:

1. Go to **Settings** → **Run Tasks**
2. Click **Create Run Task**
3. Configure:
   - **Name**: Checkmarx IaC Scan
   - **Endpoint URL**: `https://your-service-url/webhook`
   - **HMAC Key**: Your HMAC secret
   - **Stage**: Pre-apply

### 2. Associate with Workspace

1. Go to your workspace **Settings** → **Run Tasks**
2. Select your Checkmarx run task
3. Set enforcement level (Advisory/Mandatory)

## Network Architecture and Requirements

### Network Flow Overview

The HCP Terraform Checkmarx One integration involves three main components that communicate over HTTPS:

```
┌─────────────────┐    HTTPS/443     ┌─────────────────┐    HTTPS/443    ┌─────────────────┐
│                 │ ──────────────→  │                 │ ──────────────→ │                 │
│ HCP Terraform   │                  │ Listener Service│                 │ Checkmarx One   │
│                 │ ←────────────────│                 │ ←──────────────── │                 │
└─────────────────┘    HTTPS/443     └─────────────────┘    HTTPS/443    └─────────────────┘
  app.terraform.io                     your-listener.com                   *.checkmarx.net
```

### Detailed Network Flows

#### 1. HCP Terraform → Listener Service

**Initial Run Task Trigger:**
```
Source: app.terraform.io (HCP Terraform)
Destination: your-listener-service:8080
Protocol: HTTPS (TLS 1.2+)
Method: POST /webhook
Authentication: HMAC-SHA256 signature
Payload: Run task configuration with workspace details
```

**Configuration Download Request:**
```
Source: your-listener-service
Destination: app.terraform.io:443
Protocol: HTTPS (TLS 1.2+)
Method: GET /api/v2/configuration-versions/{id}/download
Authentication: Bearer token (from run task payload)
Content: Terraform workspace archive (tar.gz)
```

**Status Callbacks:**
```
Source: your-listener-service
Destination: app.terraform.io:443
Protocol: HTTPS (TLS 1.2+)
Method: PATCH /api/v2/task-results/{id}/callback
Authentication: Bearer token (from run task payload)
Content: Scan status updates and results
```

#### 2. Listener Service → Checkmarx One

**Authentication (API Key):**
```
Source: your-listener-service
Destination: {region}.iam.checkmarx.net:443
Protocol: HTTPS (TLS 1.2+)
Method: POST /auth/realms/{tenant}/protocol/openid-connect/token
Authentication: API Key
Response: Bearer access token
```

**Authentication (OAuth2):**
```
Source: your-listener-service
Destination: {region}.iam.checkmarx.net:443
Protocol: HTTPS (TLS 1.2+)
Method: POST /auth/realms/{tenant}/protocol/openid-connect/token
Authentication: Client credentials (client_id/client_secret)
Response: Bearer access token
```

**File Upload:**
```
Source: your-listener-service
Destination: {region}.ast.checkmarx.net:443
Protocol: HTTPS (TLS 1.2+)
Method: POST /api/uploads
Authentication: Bearer token
Content: Terraform archive (ZIP format)
Response: Upload URL (S3 presigned URL)
```

**S3 Upload (via presigned URL):**
```
Source: your-listener-service
Destination: *.s3.{aws-region}.amazonaws.com:443
Protocol: HTTPS (TLS 1.2+)
Method: PUT {presigned-url}
Authentication: Embedded in URL
Content: Terraform archive (ZIP format)
```

**Scan Submission:**
```
Source: your-listener-service
Destination: {region}.ast.checkmarx.net:443
Protocol: HTTPS (TLS 1.2+)
Method: POST /api/scans
Authentication: Bearer token
Content: Scan configuration with upload URL
Response: Scan ID
```

**Scan Status Polling:**
```
Source: your-listener-service
Destination: {region}.ast.checkmarx.net:443
Protocol: HTTPS (TLS 1.2+)
Method: GET /api/scans/{scan-id}
Authentication: Bearer token
Frequency: Every 30 seconds
Response: Scan status and results
```

**Results Retrieval:**
```
Source: your-listener-service
Destination: {region}.ast.checkmarx.net:443
Protocol: HTTPS (TLS 1.2+)
Method: GET /api/kics-results?scan-id={scan-id}
Authentication: Bearer token
Response: Detailed scan findings
```

### Network Requirements

#### Required Outbound Connections

| Destination | Port | Protocol | Purpose | Required |
|-------------|------|----------|---------|----------|
| `app.terraform.io` | 443 | HTTPS | HCP Terraform API | Yes |
| `*.checkmarx.net` | 443 | HTTPS | Checkmarx One API | Yes |
| `*.iam.checkmarx.net` | 443 | HTTPS | Checkmarx authentication | Yes |
| `*.s3.*.amazonaws.com` | 443 | HTTPS | File upload storage | Yes |
| DNS servers | 53 | DNS | Name resolution | Yes |

#### Required Inbound Connections

| Source | Port | Protocol | Purpose | Required |
|--------|------|----------|---------|----------|
| `app.terraform.io` | 8080 | HTTPS | Run task webhooks | Yes |
| Load balancer/proxy | 8080 | HTTP/HTTPS | Health checks | Optional |
| Monitoring systems | 8080 | HTTP | Metrics endpoint | Optional |

### Firewall Configuration

#### Outbound Rules (Egress)

```bash
# HCP Terraform API
allow tcp from listener-service to app.terraform.io port 443

# Checkmarx One API endpoints by region
allow tcp from listener-service to ast.checkmarx.net port 443        # US
allow tcp from listener-service to us.ast.checkmarx.net port 443     # US2
allow tcp from listener-service to eu.ast.checkmarx.net port 443     # EU
allow tcp from listener-service to eu-2.ast.checkmarx.net port 443   # EU2
allow tcp from listener-service to deu.ast.checkmarx.net port 443    # Germany
allow tcp from listener-service to anz.ast.checkmarx.net port 443    # ANZ
allow tcp from listener-service to ind.ast.checkmarx.net port 443    # India
allow tcp from listener-service to sng.ast.checkmarx.net port 443    # Singapore
allow tcp from listener-service to mea.ast.checkmarx.net port 443    # UAE

# Checkmarx authentication endpoints
allow tcp from listener-service to iam.checkmarx.net port 443        # US
allow tcp from listener-service to us.iam.checkmarx.net port 443     # US2
allow tcp from listener-service to eu.iam.checkmarx.net port 443     # EU
allow tcp from listener-service to eu-2.iam.checkmarx.net port 443   # EU2
allow tcp from listener-service to deu.iam.checkmarx.net port 443    # Germany
allow tcp from listener-service to anz.iam.checkmarx.net port 443    # ANZ
allow tcp from listener-service to ind.iam.checkmarx.net port 443    # India
allow tcp from listener-service to sng.iam.checkmarx.net port 443    # Singapore
allow tcp from listener-service to mea.iam.checkmarx.net port 443    # UAE

# AWS S3 for file uploads (all regions where Checkmarx operates)
allow tcp from listener-service to *.s3.us-east-1.amazonaws.com port 443
allow tcp from listener-service to *.s3.us-west-2.amazonaws.com port 443
allow tcp from listener-service to *.s3.eu-west-1.amazonaws.com port 443
allow tcp from listener-service to *.s3.eu-central-1.amazonaws.com port 443
allow tcp from listener-service to *.s3.ap-southeast-2.amazonaws.com port 443
allow tcp from listener-service to *.s3.ap-south-1.amazonaws.com port 443
allow tcp from listener-service to *.s3.ap-southeast-1.amazonaws.com port 443
allow tcp from listener-service to *.s3.me-south-1.amazonaws.com port 443

# DNS resolution
allow udp from listener-service to dns-servers port 53
allow tcp from listener-service to dns-servers port 53
```

#### Inbound Rules (Ingress)

```bash
# HCP Terraform webhooks
allow tcp from app.terraform.io to listener-service port 8080

# Health checks and monitoring
allow tcp from load-balancer to listener-service port 8080
allow tcp from monitoring-system to listener-service port 8080
```

### Proxy Configuration

#### Environment Variables

```bash
# Standard proxy settings
export HTTPS_PROXY=https://proxy.company.com:8080
export HTTP_PROXY=http://proxy.company.com:8080
export NO_PROXY=localhost,127.0.0.1,internal.company.com

# Go-specific proxy settings
export GOPROXY=https://proxy.golang.org,direct
export GOSUMDB=sum.golang.org
```

#### Configuration File

```yaml
# config.yaml
proxy_url: "https://proxy.company.com:8080"
proxy_username: "proxy-user"
proxy_password: "proxy-password"
no_proxy_hosts:
  - "localhost"
  - "127.0.0.1" 
  - "*.internal.company.com"
```

#### Proxy Authentication

The service supports:
- **Basic Authentication**: Username/password
- **NTLM Authentication**: Windows domain authentication
- **Certificate Authentication**: Client certificate authentication

### DNS Requirements

#### Required DNS Resolution

```bash
# HCP Terraform
app.terraform.io → A/AAAA records

# Checkmarx One API (region-specific)
ast.checkmarx.net → A/AAAA records
us.ast.checkmarx.net → A/AAAA records
eu.ast.checkmarx.net → A/AAAA records
deu.ast.checkmarx.net → A/AAAA records
# ... (all regional endpoints)

# Checkmarx authentication
iam.checkmarx.net → A/AAAA records
us.iam.checkmarx.net → A/AAAA records
eu.iam.checkmarx.net → A/AAAA records
# ... (all regional auth endpoints)

# AWS S3 (for file uploads)
*.s3.amazonaws.com → CNAME records
s3.us-east-1.amazonaws.com → A/AAAA records
s3.eu-central-1.amazonaws.com → A/AAAA records
# ... (regional S3 endpoints)
```

### Security Zones and Network Isolation

#### Recommended Network Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                          DMZ / Edge Zone                            │
│  ┌─────────────────┐    ┌─────────────────┐                        │
│  │   Load Balancer │    │  Web Application│                        │
│  │    (Ingress)    │    │    Firewall     │                        │
│  └─────────────────┘    └─────────────────┘                        │
└─────────────────────────────────────────────────────────────────────┘
                                   │
                                   │ Port 8080 (filtered)
                                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        Application Zone                             │
│  ┌─────────────────┐    ┌─────────────────┐                        │
│  │ HCP Terraform   │    │   Container     │                        │
│  │ Listener Service│    │   Platform      │                        │
│  └─────────────────┘    └─────────────────┘                        │
└─────────────────────────────────────────────────────────────────────┘
                                   │
                                   │ HTTPS/443 (outbound only)
                                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Internet / Cloud                           │
│  ┌─────────────────┐    ┌─────────────────┐                        │
│  │ HCP Terraform   │    │ Checkmarx One   │                        │
│  │ app.terraform.io│    │ *.checkmarx.net │                        │
│  └─────────────────┘    └─────────────────┘                        │
└─────────────────────────────────────────────────────────────────────┘
```

#### Zone Isolation Rules

**DMZ Zone:**
- Accept inbound HTTPS from internet (port 443)
- Forward webhook requests to application zone (port 8080)
- Block all other traffic between zones

**Application Zone:**
- Accept filtered inbound from DMZ only
- Allow outbound HTTPS to approved destinations
- Block all lateral movement
- Monitor and log all connections

#### Certificate Requirements

```bash
# TLS certificates needed
1. Service certificate for HTTPS endpoints
2. CA certificates for validating external APIs
3. Client certificates for proxy authentication (if required)

# Certificate validation
- Verify Checkmarx One certificates against public CA
- Verify HCP Terraform certificates against public CA
- Implement certificate pinning for critical connections
```

## API Endpoints

- `POST /webhook` - HCP Terraform run task webhook
- `GET /health` - Health check endpoint
- `GET /ready` - Readiness check endpoint
- `GET /metrics` - Prometheus metrics (if enabled)

## Development

### Prerequisites

- Go 1.22+
- Checkmarx One account with API access
- HCP Terraform organization

### Project Structure

```
├── cmd/main.go                 # Main application entry point
├── api/payload.go              # HCP Terraform payload structures
├── internal/
│   ├── config/                 # Configuration management
│   ├── checkmarx/              # Checkmarx One integration
│   ├── terraform/              # Terraform configuration parsing
│   └── httpclient/             # HTTP client utilities
├── k8s/                        # Kubernetes/OpenShift manifests
├── docs/                       # Documentation
└── Dockerfile                  # Container build definition
```

### Testing

```bash
# Run unit tests
go test ./...

# Run with coverage
go test -cover ./...

# Run integration tests
go test -tags=integration ./...
```

### Building

```bash
# Build binary
go build -o hcp-checkmarx-listener ./cmd

# Build container
./build-container.sh --tag dev

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o hcp-checkmarx-listener-linux ./cmd
```

## Security

### Container Security

- **Non-root execution**: Runs as UID 1001
- **Read-only filesystem**: Application directory is read-only
- **Minimal base image**: Uses Red Hat UBI minimal for OpenShift
- **No unnecessary capabilities**: All Linux capabilities dropped
- **Security scanning**: Built-in vulnerability scanning support

### Network Security

- **TLS support**: HTTPS endpoints with proper certificate validation
- **HMAC validation**: Request signature verification
- **Proxy support**: Corporate proxy configuration

### Secrets Management

- **Encrypted configuration**: AES-GCM encryption for sensitive data
- **Environment variables**: Secure secret injection
- **Kubernetes secrets**: Native secret management in container deployments

## Monitoring and Observability

### Metrics

Prometheus metrics available at `/metrics`:

- HTTP request duration and count
- Checkmarx API response times
- Scan processing metrics
- Error rates and types

### Logging

Structured JSON logging with configurable levels:

```json
{
  "level": "info",
  "timestamp": "2024-01-15T10:30:00Z",
  "message": "Scan completed successfully",
  "scan_id": "12345",
  "project": "my-project",
  "findings": 5
}
```

### Health Checks

- **Liveness**: `/health` - Application health status
- **Readiness**: `/ready` - Service readiness for traffic
- **Startup**: Configurable startup probe for container orchestrators

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Code Style

- Follow standard Go formatting (`go fmt`)
- Use `golangci-lint` for linting
- Write comprehensive tests for new features
- Update documentation for API changes

## Documentation

- [Network Requirements and Architecture](docs/network-requirements.md) - Complete network configuration guide with firewall rules, proxy setup, and troubleshooting
- [Configuration Reference](docs/configuration.md) - Complete configuration options including Terraform variables.tf integration
- [OpenShift Deployment Guide](docs/openshift-deployment.md) - Complete OpenShift setup with dynamic configuration examples
- [Container Registry Setup](docs/container-registry.md) - Registry configuration and authentication
- [API Documentation](docs/api.md) - REST API reference with Terraform variables processing details

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

For issues and questions:

1. Check [existing issues](https://github.com/agehring/hcp-checkmarx-listener/issues)
2. Create a new issue with detailed information
3. For security issues, please email security@example.com

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history and changes.