# HCP Terraform Run Task Integration for Checkmarx One - Configuration Guide

## Overview

This guide provides detaile| Setting | Required | Description | Default |
|-------|----------|-------------|---------|
| `checkmarx_region` | Yes | Checkmarx One region code (US, EU, etc.) | - |
| `checkmarx_project_id` | No* | Default Checkmarx One project ID | - |
| `checkmarx_token` | Yes** | API token (if using API key auth) | - |
| `checkmarx_tenant` | Yes*** | Tenant name (if using OAuth2) | - |
| `checkmarx_client_id` | Yes*** | OAuth2 client ID (if using OAuth2) | - |
| `checkmarx_client_secret` | Yes*** | OAuth2 client secret (if using OAuth2) | - |
| `listen_port` | No | Port for the service to listen on | 80 |
| `proxy_url` | No | HTTPS proxy URL for outbound requests | - |
| `break_deployment` | No | Break deployment on policy violation | false |
| `hmac_key` | No | HMAC key for request authentication | - |

*Optional if using dynamic project ID extraction from Terraform config  
**Required for API key authentication  
***Required for OAuth2 authenticationfiguration instructions for integrating HCP Terraform with Checkmarx One using the run task integration service.

## Prerequisites

- HCP Terraform Cloud/Enterprise account with run task capabilities
- Checkmarx One account with IaC scanning ### Advanced Configuration

### Dynamic Project Selection

The service supports dynamic project ID selection per Terraform workspace using three methods:

1. **Variable Declaration**: Define `checkmarx_project_id` variable with default value
2. **Resource Tags**: Add `checkmarx_project` tag to any resource
3. **Comments**: Add `# checkmarx-project: PROJECT_ID` comment anywhere in your .tf files

The service will scan the downloaded Terraform configuration and use the found project ID, falling back to the configured default if none is found.

### Custom Scan Configurationbled
- Docker or Go runtime environment
- Network connectivity to both HCP Terraform and Checkmarx One

## Checkmarx One Configuration

### 1. Obtain Checkmarx One Credentials

You have two authentication options:

#### Option A: API Key Authentication
1. Log into your Checkmarx One portal
2. Navigate to **Settings** → **API Keys**
3. Create a new API key with IaC scanning permissions
4. Note your API key for configuration

#### Option B: OAuth2 Authentication
1. Log into your Checkmarx One portal
2. Navigate to **Settings** → **OAuth Applications**
3. Create a new OAuth application
4. Note the following for configuration:
   - Client ID
   - Client Secret
   - Tenant name

### 2. Identify Your Checkmarx One Region

Determine your Checkmarx One region from the following list:

| Region | Code | Base URL | Auth URL |
|--------|------|----------|----------|
| US | US | https://ast.checkmarx.net | https://iam.checkmarx.net |
| US2 | US2 | https://us.ast.checkmarx.net | https://us.iam.checkmarx.net |
| EU | EU | https://eu.ast.checkmarx.net | https://eu.iam.checkmarx.net |
| EU2 | EUS | https://eu-2.ast.checkmarx.net | https://eu-2.iam.checkmarx.net |
| Germany | DEU | https://deu.ast.checkmarx.net | https://deu.iam.checkmarx.net |
| ANZ | ANZ | https://anz.ast.checkmarx.net | https://anz.iam.checkmarx.net |
| India | IND | https://ind.ast.checkmarx.net | https://ind.iam.checkmarx.net |
| Singapore | SNG | https://sng.ast.checkmarx.net | https://sng.iam.checkmarx.net |
| UAE | MEA | https://mea.ast.checkmarx.net | https://mea.iam.checkmarx.net |
| Israel | IL | https://gov-il.ast.checkmarx.net | https://gov-il.iam.checkmarx.net |

### 3. Get Project ID

You have two options for specifying the Checkmarx project ID:

#### Option A: Static Configuration (Default)
1. Navigate to your Checkmarx One project
2. From the project settings or URL, note the Project ID
3. Set this in the service configuration (see Configuration Fields below)

#### Option B: Dynamic Per-Workspace (Recommended)
Configure the project ID directly in your Terraform configuration using one of these methods:

**Method 1: Terraform Variable**
```hcl
variable "checkmarx_project_id" {
  description = "Checkmarx One project ID for IaC scanning"
  type        = string
  default     = "your-project-id-here"
}
```

**Method 2: Resource Tags**
```hcl
resource "aws_s3_bucket" "example" {
  bucket = "example-bucket"
  
  tags = {
    checkmarx_project = "your-project-id-here"
    Environment       = "production"
  }
}
```

**Method 3: Comment Annotation**
```hcl
# checkmarx-project: your-project-id-here
resource "aws_instance" "web" {
  ami           = "ami-0c55b159cbfafe1d0"
  instance_type = "t2.micro"
}
```

The service will automatically extract the project ID from your Terraform configuration during each run, allowing different workspaces to use different Checkmarx projects.

## Service Configuration

### Configuration Options

The service supports multiple configuration methods (in order of precedence):
1. Command-line flags
2. Environment variables
3. Configuration file

### Configuration Fields

| Field | Required | Description | Default |
|-------|----------|-------------|---------|
| `checkmarx_region` | Yes | Checkmarx One region code (US, EU, etc.) | - |
| `checkmarx_project_id` | No* | Default Checkmarx One project ID | - |
| `checkmarx_token` | Yes** | API token (if using API key auth) | - |
| `checkmarx_tenant` | Yes*** | Tenant name (if using OAuth2) | - |
| `checkmarx_client_id` | Yes*** | OAuth2 client ID (if using OAuth2) | - |
| `checkmarx_client_secret` | Yes*** | OAuth2 client secret (if using OAuth2) | - |
| `listen_port` | No | Port for the service to listen on | 80 |
| `proxy_url` | No | HTTPS proxy URL for outbound requests | - |
| `break_deployment` | No | Break deployment on policy violation | false |
| `tls_enabled` | No | Enable TLS/HTTPS server | false |
| `tls_cert_file` | No | Path to TLS certificate file | - |
| `tls_key_file` | No | Path to TLS private key file | - |
| `tls_self_signed` | No | Generate self-signed certificate if cert files not provided | false |

*Optional if using dynamic project ID extraction from Terraform config  
**Required for API key authentication  
***Required for OAuth2 authentication

### Method 1: Configuration File

Create a `config.json` file:

```json
{
  "checkmarx_region": "US",
  "checkmarx_project_id": "your-project-id",
  "checkmarx_token": "your-api-key",
  "listen_port": 80,
  "proxy_url": "",
  "break_deployment": true,
  "hmac_key": "your-hmac-key-here"
}
```

For TLS/HTTPS with self-signed certificate:

```json
{
  "checkmarx_region": "US",
  "checkmarx_project_id": "your-project-id",
  "checkmarx_token": "your-api-key",
  "listen_port": 443,
  "tls_enabled": true,
  "tls_self_signed": true
}
```

For TLS/HTTPS with existing certificate files:

```json
{
  "checkmarx_region": "US",
  "checkmarx_project_id": "your-project-id",
  "checkmarx_token": "your-api-key",
  "listen_port": 443,
  "tls_enabled": true,
  "tls_cert_file": "/path/to/certificate.crt",
  "tls_key_file": "/path/to/private.key"
}
```

For OAuth2 authentication:

```json
{
  "checkmarx_region": "US",
  "checkmarx_project_id": "your-project-id",
  "checkmarx_tenant": "your-tenant",
  "checkmarx_client_id": "your-client-id",
  "checkmarx_client_secret": "your-client-secret",
  "listen_port": 80,
  "proxy_url": "",
  "break_deployment": true,
  "hmac_key": "your-hmac-key-here"
}

```json
{
  "checkmarx_region": "EU",
  "checkmarx_project_id": "your-project-id",
  "checkmarx_tenant": "your-tenant-name",
  "checkmarx_client_id": "your-client-id",
  "checkmarx_client_secret": "your-client-secret",
  "listen_port": 80,
  "break_deployment": true
}
```

### Method 2: Environment Variables

Set the following environment variables:

```bash
# Required
export CHECKMARX_REGION="US"
export CHECKMARX_PROJECT_ID="your-project-id"

# For API Key Authentication
export CHECKMARX_TOKEN="your-api-key"

# For OAuth2 Authentication
export CHECKMARX_TENANT="your-tenant-name"
export CHECKMARX_CLIENT_ID="your-client-id"
export CHECKMARX_CLIENT_SECRET="your-client-secret"

# Optional
export LISTEN_PORT="80"
export PROXY_URL="https://proxy.company.com:8080"
export BREAK_DEPLOYMENT="true"
export HMAC_KEY="your-hmac-key-here"

# TLS Configuration
export TLS_ENABLED="true"
export TLS_CERT_FILE="/path/to/certificate.crt"
export TLS_KEY_FILE="/path/to/private.key"
export TLS_SELF_SIGNED="true"
```

### Method 3: Command-Line Flags

```bash
./main \
  --checkmarx-region="US" \
  --checkmarx-project-id="your-project-id" \
  --checkmarx-token="your-api-key" \
  --listen-port=80 \
  --break-deployment=true \
  --hmac-key="your-hmac-key-here"
```

### Secrets File Configuration

For enhanced security, sensitive values can be stored in an encrypted secrets file:

1. Create `checkmarx.secrets.json` with sensitive data:

```json
{
  "auth_type": "api_key",
  "api_key": "your-api-key"
}
```

Or for OAuth2:

```json
{
  "auth_type": "oauth2",
  "tenant": "your-tenant-name",
  "client_id": "your-client-id",
  "client_secret": "your-client-secret"
}
```

2. The service will automatically encrypt this file on first run
3. Set the secrets file path via environment variable: `CHECKMARX_SECRETS_FILE=checkmarx.secrets.json`

## HCP Terraform Configuration

### 1. Deploy the Service

Deploy the service in an environment accessible by HCP Terraform:

#### Using Docker

```bash
# Build the container
docker build -t hcp-checkmarx-listener .

# Run with environment variables
docker run -d \
  -p 80:80 \
  -e CHECKMARX_REGION="US" \
  -e CHECKMARX_PROJECT_ID="your-project-id" \
  -e CHECKMARX_TOKEN="your-api-key" \
  --name hcp-checkmarx-listener \
  hcp-checkmarx-listener
```

#### Using Go Binary

```bash
# Build the binary
go build -o main ./cmd

# Run with configuration file
./main --config=config.json
```

### 2. Configure Run Task in HCP Terraform

1. **Access HCP Terraform Settings**:
   - Navigate to your HCP Terraform organization
   - Go to **Settings** → **Run Tasks**

2. **Create New Run Task**:
   - Click **"Create run task"**
   - Provide the following details:

   | Field | Value |
   |-------|-------|
   | **Name** | Checkmarx IaC Scanner |
   | **Endpoint URL** | `http://your-service-host:80/api/run-task` |
   | **HMAC Key** | Your 64-character HMAC key (optional but recommended) |
   | **Enabled** | ✓ |

3. **Configure Run Task Association**:
   - Navigate to the workspace where you want IaC scanning
   - Go to **Settings** → **Run Tasks**
   - Click **"Add run task"**
   - Select your created run task
   - Choose enforcement level:
     - **Advisory**: Results displayed, but won't block runs
     - **Mandatory**: Failed scans will block runs
   - Select stage: **Pre-apply** (required - only stage supported)

### 3. Network Configuration

Ensure the following network connectivity:

#### Outbound from Service:
- HCP Terraform: `*.terraform.io` (port 443)
- Checkmarx One: Your region's base URL (port 443)
- If using proxy: Your proxy server

#### Inbound to Service:
- HCP Terraform agents: Your service endpoint (port 80)

### 4. Proxy Configuration (Optional)

If your environment requires a proxy for outbound internet access:

1. **Set proxy in configuration**:
   ```json
   {
     "proxy_url": "https://proxy.company.com:8080"
   }
   ```

2. **Or set environment variable**:
   ```bash
   export HTTPS_PROXY="https://proxy.company.com:8080"
   ```

## Testing the Integration

### 1. Test Service Health

```bash
curl -i http://your-service-host:80/health
```

Expected: HTTP/1.1 200 OK and a JSON body:

```json
{"status":"ok"}
```

### 2. Test with Terraform Run

1. Create a Terraform configuration with IaC resources
2. Run `terraform plan` in HCP Terraform
3. Monitor the run for the run task execution
4. Check run task results in the HCP Terraform UI

### 3. Verify Checkmarx Integration

1. Check service logs for scan submission
2. Verify scan appears in Checkmarx One console
3. Confirm policy check results are reported back to HCP Terraform

## Troubleshooting

### Common Issues

#### Connection Errors
- Verify network connectivity to both services
- Check proxy configuration if applicable
- Ensure correct region URLs are being used

#### Authentication Errors
- Verify API keys/OAuth credentials are correct
- Check token expiration for OAuth2
- Ensure proper permissions in Checkmarx One

#### Policy Errors
- Verify project ID is correct
- Check if policies are configured in Checkmarx One
- Ensure project has IaC scanning enabled

### Log Analysis

The service provides structured logging for troubleshooting:

```bash
# View service logs
docker logs hcp-checkmarx-listener

# Or if running binary directly
./main 2>&1 | tee service.log
```

Look for:
- Request receipts from HCP Terraform
- Checkmarx scan submissions and completions
- Policy check results
- Callback status updates

### Support Information

For additional support:
- Service logs with debug information
- HCP Terraform run task configuration
- Checkmarx One project and policy settings
- Network configuration details

## Security Considerations

### HMAC Authentication

For enhanced security, configure HMAC authentication to validate requests from HCP Terraform:

1. **Generate HMAC Key**: Create a secure random key for request signing:
   ```bash
   # Using OpenSSL
   openssl rand -hex 32
   
   # Using Python
   python3 -c "import secrets; print(secrets.token_hex(32))"
   ```

2. **Configure the Service**: Add the HMAC key to your configuration:
   ```json
   {
     "hmac_key": "your-64-character-hex-key-here"
   }
   ```

3. **Configure HCP Terraform**: Set the HMAC key in your run task configuration:
   - In HCP Terraform, edit your run task
   - Set the **HMAC Key** field to the same value
   - HCP Terraform will sign requests with `X-TFC-Task-Signature` header

4. **Security Benefits**:
   - Prevents unauthorized requests to your service
   - Ensures request integrity and authenticity
   - Required for production deployments

**Note**: HMAC authentication is optional but strongly recommended for production use.

### Additional Security Measures

1. **Secrets Management**: Use encrypted secrets file for production
2. **Network Security**: Deploy in secure network environment with proper firewall rules
3. **Access Control**: Limit access to configuration files and service endpoints
4. **TLS/SSL**: Deploy behind reverse proxy with TLS termination
5. **Monitoring**: Implement logging and monitoring for security events
6. **Updates**: Keep service updated with latest security patches

## Advanced Configuration

### Custom Scan Configuration

The service submits scans with default parameters. For custom scan configuration, modify the scan submission in the code or extend the configuration system.

### Multiple Projects

To support multiple Checkmarx projects, deploy separate service instances or extend the service to accept project ID from HCP Terraform metadata.

### High Availability

For production use:
- Deploy multiple service instances behind a load balancer
- Use container orchestration (Kubernetes, Docker Swarm)
- Implement health checks and auto-restart capabilities

## Troubleshooting

### Common Issues

1. **"Unsupported stage" Error**:
   - Ensure your run task is configured for **Pre-apply** or **Post-plan** stage
   - The service supports both pre_apply and post_plan stages
   - Other stages are ignored with a 200 OK response
   - Check HCP Terraform workspace run task settings

2. **"Missing signature" or "Invalid signature" Errors**:
   - Verify HMAC key matches between service config and HCP Terraform
   - Ensure the key is exactly 64 hexadecimal characters
   - Check that the run task HMAC key field is configured in HCP Terraform

3. **Connection Timeouts**:
   - Verify network connectivity from HCP Terraform to your service
   - Check firewall rules and security groups
   - Ensure your service is listening on the correct port

4. **Scan Submission Failures**:
   - Verify Checkmarx One credentials and region configuration
   - Check project ID exists and is accessible
   - Review proxy settings if using corporate proxy

### Logs and Monitoring

- Service logs include request source IP and operation details
- Security events (HMAC validation failures) are logged separately
- Monitor for unauthorized access attempts and configuration errors