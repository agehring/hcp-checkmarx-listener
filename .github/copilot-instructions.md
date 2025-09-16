# GitHub Copilot Instructions for Terraform Listener for Checkmarx IaC Scanner and Policy Check

## General GoLang Guidelines

*   **Code Style:** Follow standard Go formatting (`go fmt`) and linting (`golangci-lint`) conventions.
*   **Error Handling:** Always handle errors explicitly. Do not ignore errors.
*   **Dependency Management:** Use Go Modules for dependency management.
*   **Testing:** Write comprehensive unit and integration tests for all new features and bug fixes.
*   **Concurrency:** Use goroutines and channels appropriately for concurrent operations.

## Dockerfile Conventions

*   **Multi-stage Builds:** Utilize multi-stage builds to create lean and efficient Docker images.
*   **Base Image:** Use official GoLang base images (e.g., `golang:1.22-alpine` or `golang:1.22-slim`).
*   **Dependencies:** Install only necessary system dependencies in the build stage.
*   **Working Directory:** Set the working directory to `/app` inside the container.
*   **Entrypoint/CMD:** Define appropriate `ENTRYPOINT` and `CMD` instructions for running the Go application.
*   **Environment Variables:** Define necessary environment variables for the application within the Dockerfile or through `docker-compose.yml`.

## Project Specifics

*   **Build Process:** The Go application is built using `go build -o /app/main .` in the build stage.
*   **Application Entrypoint:** The final Docker image should execute `/app/main`.
# GitHub Copilot Instructions for HCP Terraform Run Task Integration for Checkmarx One

## Project Overview

A Go-based web service that integrates HCP Terraform with Checkmarx One for Infrastructure as Code (IaC) security scanning. This service acts as a run task handler, processing Terraform configurations and submitting them to Checkmarx One for KICS-based IaC scanning.

## General GoLang Guidelines

*   **Code Style:** Follow standard Go formatting (`go fmt`) and linting (`golangci-lint`) conventions.
*   **Error Handling:** Always handle errors explicitly. Do not ignore errors.
*   **Dependency Management:** Use Go Modules for dependency management.
*   **Testing:** Write comprehensive unit and integration tests for all new features and bug fixes.
*   **Concurrency:** Use goroutines and channels appropriately for concurrent operations.
*   **HTTP Client:** Use the centralized `internal/httpclient` package with 60s timeout for all outbound requests.

## Architecture and Project Structure

```
/
├── cmd/main.go                      # Main application entry point
├── api/payload.go                   # HCP Terraform payload structures
├── internal/
│   ├── config/config.go            # Configuration management
│   ├── secrets/secrets.go          # Encrypted secrets handling
│   ├── httpclient/httpclient.go    # Centralized HTTP client
│   └── checkmarx/
│       ├── scan.go                 # Checkmarx scan operations
│       └── policy.go               # Policy violation checking
└── go.mod
```

## Configuration System

*   **Multi-source Configuration:** Supports configuration via file, environment variables, and CLI flags.
*   **Region-based URLs:** Automatically maps region codes (US, EU, ANZ, etc.) to appropriate Checkmarx One endpoints.
*   **Proxy Support:** Configurable HTTPS proxy support via `proxy_url` setting or `HTTPS_PROXY` environment variable.
*   **Secrets Management:** Encrypted secrets file with automatic plaintext-to-encrypted conversion.

### Supported Checkmarx One Regions

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

## HCP Terraform Integration

*   **Run Task API Compliance:** Fully compliant with HCP Terraform Run Task API specification.
*   **Pre-apply Stage Only:** Only accepts and processes run tasks with "pre_apply" stage.
*   **Progressive Status Updates:** Sends status updates during scan lifecycle ("running", "passed", "failed").
*   **Rich Outcomes:** Detailed outcome objects with Severity and Status tags for optimal UI display.
*   **Authentication:** Uses access token from run task payload for callback authentication.
*   **Payload Structure:** Handles complete run task payload including capabilities, callbacks, and metadata.

### Run Task Flow

1. Receive POST request with run task payload
2. Validate access token and stage (must be "pre_apply")
3. Enqueue job for processing
4. Download Terraform configuration archive
5. Submit to Checkmarx One for IaC scan
6. Poll for scan completion
7. Retrieve scan results and findings
8. Check policy violations
9. Send detailed outcomes to HCP Terraform callback

## Checkmarx One Integration

*   **IaC Scanning:** Submits ZIP archives to Checkmarx One KICS engine.
*   **Scan Management:** Polls for completion and retrieves detailed results.
*   **Policy Enforcement:** Checks policy violations and supports break-build scenarios.
*   **Authentication:** Supports both API key and OAuth2 authentication methods.
*   **Finding Details:** Retrieves comprehensive finding information including severity, remediation, and metadata.

### API Endpoints Used

*   **Scan Submission:** `POST /api/scans`
*   **Scan Status:** `GET /api/scans/{scanId}`
*   **Scan Results:** `GET /api/kics-results/`
*   **Policy Check:** `GET /api/policy_management_service_uri/evaluation`

## Security Features

*   **Encrypted Secrets:** AES-GCM encryption for sensitive configuration data.
*   **Request Validation:** Validates access tokens and logs security events.
*   **Proxy Support:** Secure proxy configuration for enterprise environments.
*   **Timeout Controls:** 60-second timeout on all HTTP requests to prevent hanging.
*   **Sensitive Data Protection:** Avoids logging tokens, secrets, or PII.

## Dockerfile Conventions

*   **Multi-stage Builds:** Utilize multi-stage builds to create lean and efficient Docker images.
*   **Base Image:** Use official GoLang base images (e.g., `golang:1.22-alpine` or `golang:1.22-slim`).
*   **Dependencies:** Install only necessary system dependencies in the build stage.
*   **Working Directory:** Set the working directory to `/app` inside the container.
*   **Entrypoint/CMD:** Define appropriate `ENTRYPOINT` and `CMD` instructions for running the Go application.
*   **Environment Variables:** Define necessary environment variables for the application within the Dockerfile or through `docker-compose.yml`.
*   **Build Process:** The Go application is built using `go build -o /app/main .` in the build stage.
*   **Application Entrypoint:** The final Docker image should execute `/app/main`.

## Development Guidelines

*   **Error Handling:** Comprehensive error handling with appropriate HTTP status codes.
*   **Logging:** Structured logging with security-conscious message filtering.
*   **Testing:** Unit tests for all major components and integration tests for API interactions.
*   **Configuration:** All settings configurable via multiple sources (file, env, CLI).
*   **Concurrency:** Job queue with worker goroutines for asynchronous processing.

## When Generating Code

*   Prioritize clarity and readability in generated Go code.
*   Use the established project structure and patterns.
*   Follow security best practices for handling sensitive data.
*   Ensure all HTTP requests use the centralized httpclient package.
*   Include appropriate error handling and logging.
*   Follow the HCP Terraform Run Task API specification for callbacks.
*   Use the correct Checkmarx One API endpoints and response structures.
*   ** Create a group of Checkmarx functions in go that
       - Submits a scan using a zip archive for IaC using the checkmarx one APIs 
         - Documentation at (https://checkmarx.stoplight.io/docs/checkmarx-one-api-reference-guide/3w7wczsazj6pg-introduction
         - Swagger at https://ast.checkmarx.net/spec/v1
       - Checks the Policy for a violation when related to a IaC scan
*   **Map the JSON Payload to a a golnag struct, based on this example:
      {
  "payload_version": 1,
  "stage": "post_plan",
  "access_token": "",
  "capabilities": {
    "outcomes": true
  },
  "configuration_version_download_url": "https://app.terraform.io/api/v2/configuration-versions/cv-ntv3HbhJqvFzamy7/download",
  "configuration_version_id": "cv-ntv3HbhJqvFzamy7",
  "is_speculative": false,
  "organization_name": "hashicorp",
  "plan_json_api_url": "https://app.terraform.io/api/v2/plans/plan-6AFmRJW1PFJ7qbAh/json-output",
  "run_app_url": "https://app.terraform.io/app/hashicorp/my-workspace/runs/run-i3Df5to9ELvibKpQ",
  "run_created_at": "2021-09-02T14:47:13.036Z",
  "run_created_by": "username",
  "run_id": "run-i3Df5to9ELvibKpQ",
  "run_message": "Triggered via UI",
  "task_result_callback_url": "https://app.terraform.io/api/v2/task-results/5ea8d46c-2ceb-42cd-83f2-82e54697bddd/callback",
  "task_result_enforcement_level": "mandatory",
  "task_result_id": "taskrs-2nH5dncYoXaMVQmJ",
  "vcs_branch": "main",
  "vcs_commit_url": "https://github.com/hashicorp/terraform-random/commit/7d8fb2a2d601edebdb7a59ad2088a96673637d22",
  "vcs_pull_request_url": null,
  "vcs_repo_url": "https://github.com/hashicorp/terraform-random",
  "workspace_app_url": "https://app.terraform.io/app/hashicorp/my-workspace",
  "workspace_id": "ws-ck4G5bb1Yei5szRh",
  "workspace_name": "tfr_github_0",
  "workspace_working_directory": "/terraform"
}
*   **Support the HCP Terraform APIs for a Run Task Integration (https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run-tasks/run-tasks-integration)
*   **Write a function in golang that downloads a file from a URL, using an authorization header
*   **Create a secure Dockerfile that runs the web service created

## When Generating Code

*   Prioritize clarity and readability in generated Go code.
*   Suggest Dockerfile improvements for efficiency and security.
*   When generating tests, aim for good test coverage and clear test cases.
*   If suggesting API endpoints, use idiomatic Go HTTP server patterns.
