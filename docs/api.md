# API Documentation

This document describes the API endpoints and integration details for the HCP Checkmarx Listener, including the dynamic Terraform variables.tf functionality.

## Table of Contents

- [API Endpoints](#api-endpoints)
- [HCP Terraform Integration](#hcp-terraform-integration)
- [Terraform Variables Processing](#terraform-variables-processing)
- [Checkmarx One Integration](#checkmarx-one-integration)
- [Error Handling](#error-handling)

## API Endpoints

### POST /api/run-task

**Description:** HCP Terraform run task webhook endpoint that receives run task payloads and processes them for Checkmarx One scanning.

**Supported Stages:** `pre_apply`, `post_plan`  
**Note:** Unsupported stages receive a 200 OK response without processing.

**Request Headers:**
```
Content-Type: application/json
X-TFC-Task-Signature: sha256=<hmac_signature>
```

**Request Body:**
```json
{
  "payload_version": 1,
  "stage": "pre_apply",
  "access_token": "at-xxx",
  "capabilities": {
    "outcomes": true
  },
  "configuration_version_download_url": "https://app.terraform.io/api/v2/configuration-versions/cv-xxx/download",
  "configuration_version_id": "cv-xxx",
  "is_speculative": false,
  "organization_name": "my-org",
  "plan_json_api_url": "https://app.terraform.io/api/v2/plans/plan-xxx/json-output",
  "run_app_url": "https://app.terraform.io/app/my-org/my-workspace/runs/run-xxx",
  "run_created_at": "2024-01-15T10:30:00Z",
  "run_created_by": "username",
  "run_id": "run-xxx",
  "run_message": "Triggered via UI",
  "task_result_callback_url": "https://app.terraform.io/api/v2/task-results/xxx/callback",
  "task_result_enforcement_level": "mandatory",
  "task_result_id": "taskrs-xxx",
  "vcs_branch": "main",
  "vcs_commit_url": "https://github.com/org/repo/commit/abc123",
  "vcs_pull_request_url": null,
  "vcs_repo_url": "https://github.com/org/repo",
  "workspace_app_url": "https://app.terraform.io/app/my-org/my-workspace",
  "workspace_id": "ws-xxx",
  "workspace_name": "my-workspace",
  "workspace_working_directory": "/terraform"
}
```

**Alternative Example (Post-Plan Stage):**
```json
{
  "payload_version": 1,
  "stage": "post_plan",
  "access_token": "at-xyz",
  "capabilities": {
    "outcomes": true
  },
  "configuration_version_download_url": "https://app.terraform.io/api/v2/configuration-versions/cv-xyz/download",
  "run_id": "run-xyz",
  "workspace_name": "production-workspace",
  "organization_name": "my-org"
}
```

**Response:**
```json
{
  "data": {
    "type": "task-results",
    "attributes": {
      "status": "running",
      "message": "Submitted to Checkmarx for scan",
      "url": "https://your-service.com/results/task-xxx"
    }
  }
}
```

**Status Codes:**
- `200 OK`: Request accepted and processing started
- `400 Bad Request`: Invalid payload or missing required fields
- `401 Unauthorized`: Invalid HMAC signature
- `500 Internal Server Error`: Service error

## Testing the API

**Manual Testing:**
```bash
# Test the endpoint (will fail without proper Checkmarx credentials)
curl -X POST http://localhost:8080/api/run-task \
  -H "Content-Type: application/json" \
  -d '{
    "stage": "pre_apply",
    "access_token": "test-token",
    "run_id": "test-run",
    "workspace_name": "test-workspace",
    "organization_name": "test-org"
  }'
```

### GET /health

**Description:** Health check endpoint for container orchestrators and load balancers.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "checkmarx_connectivity": "ok"
}
```

**Status Codes:**
- `200 OK`: Service is healthy
- `503 Service Unavailable`: Service is unhealthy

### GET /ready

**Description:** Readiness check endpoint indicating service is ready to accept traffic.

**Response:**
```json
{
  "status": "ready",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

**Status Codes:**
- `200 OK`: Service is ready
- `503 Service Unavailable`: Service is not ready

### GET /metrics

**Description:** Prometheus metrics endpoint (if metrics are enabled).

**Response:** Prometheus-formatted metrics
```
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="POST",endpoint="/webhook",status="200"} 42

# HELP checkmarx_scans_total Total number of Checkmarx scans submitted
# TYPE checkmarx_scans_total counter
checkmarx_scans_total{status="success"} 38
checkmarx_scans_total{status="error"} 4

# HELP terraform_variables_parsed_total Total number of Terraform variables parsed
# TYPE terraform_variables_parsed_total counter
terraform_variables_parsed_total{variable="checkmarx_project_name"} 25
terraform_variables_parsed_total{variable="checkmarx_group"} 20
terraform_variables_parsed_total{variable="checkmarx_branch"} 30
```

## HCP Terraform Integration

### Run Task Flow

1. **Webhook Receipt**: Receive run task payload from HCP Terraform
2. **Validation**: Verify HMAC signature and payload structure
3. **Download**: Download Terraform configuration archive
4. **Parse**: Extract Terraform variables for Checkmarx configuration
5. **Scan**: Submit to Checkmarx One with dynamic configuration
6. **Poll**: Monitor scan progress
7. **Results**: Retrieve findings and policy violations
8. **Callback**: Send outcomes back to HCP Terraform

### Callback API

The service sends callbacks to HCP Terraform with scan results:

**Status Update:**
```json
{
  "data": {
    "type": "task-results",
    "attributes": {
      "status": "running",
      "message": "Scan in progress...",
      "url": "https://ast.checkmarx.net/CxWebClient/ScanDetails.aspx?scanId=123456"
    }
  }
}
```

**Final Results with Outcomes:**
```json
{
  "data": {
    "type": "task-results",
    "attributes": {
      "status": "passed",
      "message": "Scan completed: 5 findings detected",
      "url": "https://ast.checkmarx.net/CxWebClient/ScanDetails.aspx?scanId=123456",
      "outcomes": [
        {
          "outcome_id": "finding-1",
          "description": "High severity SQL injection vulnerability in RDS configuration",
          "body": "**File:** `modules/database/main.tf`\n**Line:** 45\n**Severity:** HIGH\n\n**Description:**\nThe RDS instance configuration allows connections from any IP address (0.0.0.0/0), which poses a security risk.\n\n**Remediation:**\nRestrict the security group to allow connections only from trusted IP ranges or VPC CIDR blocks.",
          "tags": {
            "Severity": "HIGH",
            "Status": "New",
            "Category": "Infrastructure"
          }
        }
      ]
    }
  }
}
```

## Terraform Variables Processing

### Variable Discovery

The service scans downloaded Terraform configurations for `variables.tf` files and parses them for Checkmarx-specific variables.

**Supported Variable Names:**
- `checkmarx_project_name`: Checkmarx One project name
- `checkmarx_group`: Checkmarx One group/team identifier
- `checkmarx_branch`: Branch name for scan context

### Parsing Logic

**1. File Discovery:**
```go
// Check if input is tar.gz archive or directory
if strings.HasSuffix(inputPath, ".tar.gz") {
    // Extract archive to temporary directory
    terraformDir = extractTarGz(inputPath)
}

// Recursively find all variables.tf files
files := findVariablesFiles(terraformDir)
```

**2. Archive Extraction:**
```go
// Extract tar.gz files to temporary directory
func extractTarGz(tarGzPath string) (string, error) {
    tempDir := os.MkdirTemp("", "terraform-workspace-*")
    // Extract using archive/tar and compress/gzip
    return tempDir, nil
}
```

**3. HCL Parsing:**
```go
// Parse each variables.tf file for specific variables
supportedVars := []string{
    "checkmarx_project_name",
    "checkmarx_group", 
    "checkmarx_branch"
}
```

**4. Configuration Building:**
```go
// Build final configuration with precedence and validation
type CxOneConfig struct {
    ProjectName string  // From checkmarx_project_name
    Group       string  // From checkmarx_group (auto-default if empty)
    Branch      string  // From checkmarx_branch
}
```

### Example Processing

**Input (variables.tf):**
```hcl
variable "checkmarx_project_name" {
  description = "Project for security scanning"
  type        = string
  default     = "web-application-iac"
}

variable "checkmarx_group" {
  description = "Security team identifier"
  type        = string
  default     = "Platform-Security"
}

variable "checkmarx_branch" {
  description = "Development branch"
  type        = string
  default     = "feature/database-encryption"
}

# Other variables are ignored
variable "instance_type" {
  default = "t3.micro"
}
```

**Parsed Configuration:**
```json
{
  "project_name": "web-application-iac",
  "group": "Platform-Security",
  "branch": "feature/database-encryption"
}
```

**Resulting Scan Request:**
```json
{
  "type": "upload",
  "handler": {
    "name": "kics",
    "branch": "feature/database-encryption"
  },
  "project": {
    "id": "web-application-iac"
  },
  "config": [
    {
      "type": "kics",
      "value": {
        "additionalParameters": ""
      }
    }
  ],
  "tags": {
    "hcp_task_result_id": "taskrs-xxx",
    "hcp_run_id": "run-xxx",
    "hcp_workspace": "my-workspace",
    "branch": "feature/database-encryption",
    "group": "Platform-Security"
  }
}
```

## Checkmarx One Integration

### Scan Submission

**Endpoint:** `POST /api/scans`

**Request:**
```json
{
  "type": "upload",
  "handler": {
    "name": "kics",
    "branch": "main"
  },
  "project": {
    "id": "project-id"
  },
  "config": [
    {
      "type": "kics",
      "value": {
        "additionalParameters": ""
      }
    }
  ],
  "tags": {
    "hcp_task_result_id": "taskrs-xxx",
    "hcp_run_id": "run-xxx",
    "hcp_workspace": "workspace-name",
    "branch": "feature-branch",
    "group": "team-name"
  }
}
```

### Results Retrieval

**Endpoint:** `GET /api/kics-results/`

**Query Parameters:**
- `scan-id`: The scan ID returned from submission
- `limit`: Maximum number of results (default: 100)
- `offset`: Pagination offset (default: 0)

**Response:**
```json
{
  "results": [
    {
      "id": "finding-123",
      "type": "kics",
      "severity": "HIGH",
      "status": "NEW",
      "state": "TO_VERIFY",
      "created": "2024-01-15T10:30:00Z",
      "description": "SQL injection vulnerability",
      "data": {
        "queryId": "b86b14a2-918a-44b9-8e4d-2d8e83a98f15",
        "queryName": "RDS Instance Publicly Accessible",
        "group": "AWS",
        "category": "Networking and Firewall",
        "descriptionId": "0de7ad5c-b5b7-4c4e-8774-e6b7d0c4c5e1",
        "description": "RDS instance is publicly accessible",
        "descriptionUrl": "https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_CommonTasks.Connect.html",
        "platform": "Terraform",
        "cloudProvider": "AWS",
        "cwe": "CWE-16",
        "owasp": "A6"
      },
      "vulnerabilityDetails": {
        "fileName": "modules/database/main.tf",
        "line": 45,
        "issueType": "InfoLeak",
        "value": "true",
        "expectedValue": "false"
      }
    }
  ],
  "totalCount": 5,
  "scanId": "scan-123456"
}
```

### Policy Evaluation

**Endpoint:** `GET /api/policy_management_service_uri/evaluation`

**Query Parameters:**
- `project-id`: The project ID
- `scan-id`: The scan ID

**Response:**
```json
{
  "status": "Violated",
  "rules": [
    {
      "type": "severity",
      "value": "HIGH",
      "count": 3,
      "threshold": 0,
      "violated": true
    }
  ],
  "breakBuild": true
}
```

## Error Handling

### Error Response Format

All error responses follow this structure:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid payload: missing required field 'access_token'",
    "details": {
      "field": "access_token",
      "provided": null,
      "expected": "string"
    },
    "timestamp": "2024-01-15T10:30:00Z",
    "trace_id": "trace-123456"
  }
}
```

### Error Codes

| Code | Description | HTTP Status |
|------|-------------|-------------|
| `VALIDATION_ERROR` | Request validation failed | 400 |
| `AUTHENTICATION_ERROR` | HMAC signature invalid | 401 |
| `WORKSPACE_DOWNLOAD_ERROR` | Failed to download Terraform workspace | 500 |
| `TERRAFORM_PARSE_ERROR` | Failed to parse Terraform variables | 500 |
| `CHECKMARX_API_ERROR` | Checkmarx One API error | 500 |
| `SCAN_SUBMISSION_ERROR` | Failed to submit scan | 500 |
| `SCAN_TIMEOUT_ERROR` | Scan took too long to complete | 500 |
| `POLICY_EVALUATION_ERROR` | Failed to evaluate policy | 500 |

### Terraform Variables Parsing Errors

When Terraform variables parsing fails, the service logs warnings but continues with default values:

**Archive Extraction Error:**
```json
{
  "level": "error",
  "timestamp": "2024-01-15T10:30:00Z",
  "message": "Error parsing CxOne config from Terraform, using defaults",
  "error": "failed to extract tar.gz: gzip: invalid header",
  "archive": "/tmp/hcp-config-123456.tar.gz",
  "workspace": "my-workspace",
  "run_id": "run-123456"
}
```

**Variable Parsing Warning:**
```json
{
  "level": "warn",
  "timestamp": "2024-01-15T10:30:00Z",
  "message": "Failed to parse Terraform variables, using defaults",
  "error": "invalid HCL syntax in variables.tf line 15",
  "file": "/tmp/workspace/variables.tf",
  "workspace": "my-workspace",
  "run_id": "run-123456"
}
```

**Empty Group Auto-Correction:**
```json
{
  "level": "info",
  "timestamp": "2024-01-15T10:30:00Z",
  "message": "Empty group name detected, using default: hcp-tasks",
  "workspace": "my-workspace",
  "run_id": "run-123456"
}
```

**Default Fallback Values:**
```json
{
  "project_name": "hcp-terraform-project",
  "group": "hcp-tasks",
  "branch": "main"
}
```

### Retry Logic

The service implements retry logic for:

1. **Checkmarx API calls**: 3 retries with exponential backoff
2. **Scan status polling**: Continues until timeout (10 minutes default)
3. **HCP Terraform callbacks**: 3 retries with 5-second intervals

### Monitoring and Alerting

**Key Metrics to Monitor:**
- `http_request_duration_seconds`: Request processing time
- `checkmarx_scan_success_total`: Successful scan submissions
- `checkmarx_scan_error_total`: Failed scan submissions
- `terraform_variable_parse_success_total`: Successful variable parsing
- `terraform_variable_parse_error_total`: Failed variable parsing
- `policy_violation_total`: Policy violations detected

**Recommended Alerts:**
- High error rate (>5% in 5 minutes)
- Scan timeout rate increase
- Terraform parsing failure rate increase
- Checkmarx API connectivity issues