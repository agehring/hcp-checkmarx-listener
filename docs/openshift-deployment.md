# OpenShift Deployment Guide

This guide covers deploying the HCP Checkmarx Listener on Red Hat OpenShift using CRI-O containers.

## Prerequisites

- OpenShift cluster 4.6+ with CRI-O runtime
- `oc` CLI tool configured and authenticated
- Access to a container registry (quay.io, OpenShift internal registry, etc.)
- Checkmarx One API credentials

## Quick Start

### 1. Build and Push Container Image

```bash
# Build for OpenShift internal registry
./build-container.sh \
  --registry image-registry.openshift-image-registry.svc:5000 \
  --namespace myproject \
  --tag latest \
  --push

# Or build for external registry
./build-container.sh \
  --registry quay.io \
  --namespace your-org \
  --tag v1.0.0 \
  --push
```

### 2. Deploy to OpenShift

#### Option A: All-in-One Deployment

```bash
# Deploy everything at once
oc apply -f k8s/all-in-one.yaml

# Set secrets
oc create secret generic hcp-checkmarx-secrets \
  --from-literal=checkmarx_api_key=$(echo 'your-api-key' | base64) \
  --from-literal=hmac_secret=$(echo 'your-hmac-secret' | base64) \
  -n hcp-checkmarx
```

#### Option B: Step-by-Step Deployment

```bash
# Create namespace and RBAC
oc apply -f k8s/rbac.yaml

# Create configuration
oc apply -f k8s/configmap.yaml

# Create secrets (update the secret values first)
oc apply -f k8s/configmap.yaml

# Deploy application
oc apply -f k8s/deployment.yaml

# Create service and route
oc apply -f k8s/service.yaml
```

### 3. Configure Secrets

Update the secrets with your actual values:

```bash
# Checkmarx API Key
oc patch secret hcp-checkmarx-secrets \
  -p '{"data":{"checkmarx_api_key":"'$(echo -n 'your-actual-api-key' | base64)'"}}' \
  -n hcp-checkmarx

# HMAC Secret for request validation
oc patch secret hcp-checkmarx-secrets \
  -p '{"data":{"hmac_secret":"'$(echo -n 'your-hmac-secret' | base64)'"}}' \
  -n hcp-checkmarx
```

## Configuration Options

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8080` |
| `CONFIG_FILE` | Path to config file | `/etc/config/config.yaml` |
| `CHECKMARX_API_KEY` | Checkmarx One API key | Required |
| `CHECKMARX_BASE_URL` | Checkmarx One API URL | `https://ast.checkmarx.net` |
| `CHECKMARX_AUTH_URL` | Checkmarx One Auth URL | `https://iam.checkmarx.net` |
| `HMAC_SECRET` | HMAC secret for validation | Required |

### Dynamic Terraform Configuration

The service automatically parses Terraform `variables.tf` files from HCP Terraform workspaces to extract Checkmarx One scan parameters. This allows users to customize scans per workspace.

#### Supported Terraform Variables

Users can add these variables to their `variables.tf` files:

```hcl
variable "checkmarx_project_name" {
  description = "The name of the Checkmarx One project"
  type        = string
  default     = "my-terraform-project"
}

variable "checkmarx_group" {
  description = "The Checkmarx One group or team"
  type        = string
  default     = "Infrastructure"
}

variable "checkmarx_branch" {
  description = "The branch name for the scan"
  type        = string
  default     = "main"
}
```

#### How It Works

1. **Download**: Service downloads Terraform workspace from HCP Terraform
2. **Parse**: Automatically scans all `variables.tf` files in the workspace
3. **Extract**: Retrieves default values from Checkmarx variable declarations
4. **Apply**: Uses extracted values for scan submission to Checkmarx One
5. **Fallback**: Uses service defaults if variables are not found

#### Configuration Precedence

1. **Terraform variables.tf defaults** (highest priority)
2. **ConfigMap configuration**
3. **Environment variables**
4. **Built-in defaults** (lowest priority)

### ConfigMap Configuration

Edit `k8s/configmap.yaml` to customize:

```yaml
data:
  config.yaml: |
    port: "8080"
    checkmarx_base_url: "https://ast.checkmarx.net"
    checkmarx_auth_url: "https://iam.checkmarx.net"
    checkmarx_region: "US"  # US, EU, ANZ, etc.
    use_oauth: false
    enable_policy_check: true
    break_on_policy_violation: true
    log_level: "info"
    request_timeout: "60s"
    max_workers: 5
    queue_size: 100
```

## Security Considerations

### OpenShift Security Context Constraints (SCC)

The deployment includes a custom SCC (`hcp-checkmarx-scc`) that:

- Runs containers as non-root users (UID 1000-65534)
- Uses read-only root filesystem
- Drops all Linux capabilities
- Prevents privilege escalation
- Enforces SELinux policies

### Container Security Features

- **Non-root execution**: Runs as UID 1001
- **Read-only filesystem**: Application directory is read-only
- **Minimal base image**: Uses Red Hat UBI minimal
- **No unnecessary capabilities**: All capabilities dropped
- **Security scanning**: Built with vulnerability scanning

### Network Security

- **TLS termination**: Routes terminate TLS at the edge
- **Service mesh ready**: Compatible with OpenShift Service Mesh
- **Network policies**: Can be restricted with NetworkPolicy resources

## Monitoring and Observability

### Health Checks

The deployment includes comprehensive health checks:

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5

startupProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10
  failureThreshold: 30
```

### Metrics and Logging

- **Prometheus metrics**: Available at `/metrics` endpoint
- **Structured logging**: JSON format logs for OpenShift logging
- **Distributed tracing**: Compatible with Jaeger/OpenTelemetry

### Resource Management

```yaml
resources:
  requests:
    memory: "128Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

## Scaling and High Availability

### Horizontal Pod Autoscaling

```bash
oc autoscale deployment hcp-checkmarx-listener \
  --min=2 --max=10 \
  --cpu-percent=70 \
  -n hcp-checkmarx
```

### Pod Disruption Budget

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: hcp-checkmarx-listener-pdb
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app: hcp-checkmarx-listener
```

## Troubleshooting

### Common Issues

1. **Image Pull Errors**
   ```bash
   # Check image pull secrets
   oc get pods -n hcp-checkmarx
   oc describe pod <pod-name> -n hcp-checkmarx
   ```

2. **Permission Denied**
   ```bash
   # Check SCC assignment
   oc get scc hcp-checkmarx-scc
   oc adm policy who-can use scc hcp-checkmarx-scc
   ```

3. **Secret Issues**
   ```bash
   # Verify secrets are properly base64 encoded
   oc get secret hcp-checkmarx-secrets -o yaml -n hcp-checkmarx
   ```

### Debug Commands

```bash
# Check pod logs
oc logs deployment/hcp-checkmarx-listener -n hcp-checkmarx

# Port forward for local testing
oc port-forward svc/hcp-checkmarx-listener 8080:80 -n hcp-checkmarx

# Execute into pod
oc exec -it deployment/hcp-checkmarx-listener -n hcp-checkmarx -- /bin/sh

# Check configuration
oc get configmap hcp-checkmarx-config -o yaml -n hcp-checkmarx
```

## Advanced Deployment Options

### Environment-Specific Configuration

For different environments, you can customize the ConfigMap per namespace:

**Development Environment:**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: hcp-checkmarx-config
  namespace: hcp-checkmarx-dev
data:
  config.yaml: |
    port: "8080"
    checkmarx_base_url: "https://ast.checkmarx.net"
    checkmarx_region: "US"
    checkmarx_project_id: "dev-infrastructure"
    log_level: "debug"
    policy:
      break_on_policy_violation: false  # Advisory only in dev
```

**Production Environment:**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: hcp-checkmarx-config
  namespace: hcp-checkmarx-prod
data:
  config.yaml: |
    port: "8080"
    checkmarx_base_url: "https://ast.checkmarx.net"
    checkmarx_region: "US"
    checkmarx_project_id: "prod-infrastructure"
    log_level: "info"
    policy:
      break_on_policy_violation: true  # Enforce in production
```

### Terraform Variables.tf Examples

Users can customize their scans by adding variables to their Terraform configurations:

**Basic Configuration:**
```hcl
# In your workspace's variables.tf
variable "checkmarx_project_name" {
  description = "Project name for Checkmarx scanning"
  default     = "web-app-infrastructure"
}

variable "checkmarx_group" {
  description = "Team responsible for this infrastructure"
  default     = "Platform-Engineering"
}

variable "checkmarx_branch" {
  description = "Current development branch"
  default     = "feature/new-security-group"
}
```

**Environment-Aware Configuration:**
```hcl
# Dynamic configuration based on Terraform workspace
locals {
  environment = terraform.workspace
  
  checkmarx_config = {
    dev = {
      project_name = "myapp-dev-iac"
      group        = "Development"
      branch       = "develop"
    }
    staging = {
      project_name = "myapp-staging-iac"
      group        = "QA-Team"
      branch       = "staging"
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

**Multi-Team Configuration:**
```hcl
# Team-specific scanning configuration
variable "team_name" {
  description = "Team name for resource tagging"
  type        = string
  default     = "platform"
}

variable "checkmarx_project_name" {
  description = "Checkmarx project based on team"
  type        = string
  default     = "${var.team_name}-infrastructure-security"
}

variable "checkmarx_group" {
  description = "Checkmarx group mapping"
  type        = string
  default     = title("${var.team_name}-Team")
}

variable "checkmarx_branch" {
  description = "Branch name for scan context"
  type        = string
  default     = "main"
}
```

### Using Helm (Optional)

Create a Helm chart for more complex deployments:

```bash
helm create hcp-checkmarx-listener
# Copy k8s/ manifests to templates/
# Add values.yaml for configuration
```

### GitOps with ArgoCD

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: hcp-checkmarx-listener
spec:
  project: default
  source:
    repoURL: https://github.com/your-org/hcp-checkmarx-listener
    targetRevision: HEAD
    path: k8s
  destination:
    server: https://kubernetes.default.svc
    namespace: hcp-checkmarx
```

### Multi-cluster Deployment

For deploying across multiple OpenShift clusters, consider:

- **Advanced Cluster Management (ACM)**: For policy-based deployment
- **Red Hat Advanced Cluster Security**: For security compliance
- **OpenShift GitOps**: For declarative multi-cluster management

## Maintenance

### Updates and Rollbacks

```bash
# Update image
oc set image deployment/hcp-checkmarx-listener \
  hcp-checkmarx-listener=quay.io/your-org/hcp-checkmarx-listener:v1.1.0 \
  -n hcp-checkmarx

# Rollback
oc rollout undo deployment/hcp-checkmarx-listener -n hcp-checkmarx

# Check rollout status
oc rollout status deployment/hcp-checkmarx-listener -n hcp-checkmarx
```

### Backup and Recovery

```bash
# Backup configuration
oc get configmap,secret,deployment,service,route -n hcp-checkmarx -o yaml > backup.yaml

# Restore
oc apply -f backup.yaml
```