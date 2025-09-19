# GitHub Actions CI/CD Pipeline

This repository includes comprehensive GitHub Actions workflows for building, testing, securing, and releasing both Docker and OpenShift CRI-O containers.

## Workflows

### 1. Docker Build (`docker-build.yml`)

**Triggers:**
- Push to `main`, `pre-release`, `develop` branches
- Pull requests to `main`, `pre-release`
- Tags starting with `v*`
- Manual dispatch

**Features:**
- Multi-architecture builds (linux/amd64, linux/arm64)
- Pushes to GitHub Container Registry (`ghcr.io`)
- Vulnerability scanning with Trivy
- SBOM (Software Bill of Materials) generation
- Security scan results uploaded to GitHub Security tab
- Build caching for faster builds

**Registry:** `ghcr.io/<username>/hcp-checkmarx-listener`

### 2. OpenShift Build (`openshift-build.yml`)

**Triggers:**
- Push to `main`, `pre-release`, `develop` branches
- Pull requests to `main`, `pre-release`
- Tags starting with `v*`
- Manual dispatch

**Features:**
- OpenShift-optimized container build
- Multi-architecture builds (linux/amd64, linux/arm64)
- Pushes to Quay.io registry
- OpenShift-specific labels and annotations
- Automated OpenShift deployment manifests generation
- Red Hat UBI base image for enterprise compliance
- Security context optimized for OpenShift arbitrary UIDs

**Registry:** `quay.io/<username>/hcp-checkmarx-listener-openshift`

### 3. Security & Release (`security-release.yml`)

**Triggers:**
- Tags starting with `v*`
- Manual dispatch with version increment options

**Features:**
- Security scanning with Snyk and Gosec
- Automated semantic versioning
- GitHub releases with changelog generation
- Binary releases for multiple platforms via GoReleaser
- Container image signing with Cosign
- SLSA provenance attestations
- Comprehensive release notifications

## Setup Requirements

### Repository Secrets

Add these secrets to your GitHub repository:

```bash
# For Quay.io (OpenShift registry)
QUAY_USERNAME=<your-quay-username>
QUAY_TOKEN=<your-quay-token>

# For Snyk security scanning (optional)
SNYK_TOKEN=<your-snyk-token>
```

### Registry Setup

1. **GitHub Container Registry (GHCR):**
   - Automatically configured using `GITHUB_TOKEN`
   - No additional setup required

2. **Quay.io (for OpenShift):**
   - Create account at [quay.io](https://quay.io)
   - Generate robot account or personal access token
   - Add `QUAY_USERNAME` and `QUAY_TOKEN` to repository secrets

## Usage

### Automatic Builds

Containers are automatically built on:
- Every push to main branches
- Every pull request
- Every tag creation

### Manual Release

1. Go to Actions → "Container Security and Release"
2. Click "Run workflow"
3. Select release type (patch/minor/major)
4. Click "Run workflow"

This will:
- Increment version automatically
- Create GitHub release with changelog
- Build and sign container images
- Generate security attestations

### Container Image Usage

**Docker (GitHub Container Registry):**
```bash
# Latest from main branch
docker pull ghcr.io/<username>/hcp-checkmarx-listener:latest

# Specific version
docker pull ghcr.io/<username>/hcp-checkmarx-listener:v1.0.0

# Run container
docker run -p 8080:8080 ghcr.io/<username>/hcp-checkmarx-listener:latest
```

**OpenShift (Quay.io):**
```bash
# Latest from main branch
docker pull quay.io/<username>/hcp-checkmarx-listener-openshift:latest

# Specific version
docker pull quay.io/<username>/hcp-checkmarx-listener-openshift:v1.0.0

# OpenShift deployment
oc apply -f k8s/openshift/deployment.yaml
```

## Security Features

### Container Scanning
- **Trivy:** Vulnerability scanning for both container types
- **Snyk:** Go dependency vulnerability scanning
- **Gosec:** Go security code analysis

### Supply Chain Security
- **SBOM:** Software Bill of Materials for all images
- **Cosign:** Container image signing
- **SLSA Provenance:** Build attestations
- **Multi-arch builds:** Support for amd64 and arm64

### OpenShift Security
- **Non-root user:** Runs as UID 1001
- **Arbitrary UID support:** Compatible with OpenShift security constraints
- **Security context:** Minimal privileges with security profiles
- **Red Hat UBI:** Enterprise-grade base image

## Container Verification

All container images are signed and can be verified:

```bash
# Install cosign
go install github.com/sigstore/cosign/v2/cmd/cosign@latest

# Verify Docker image signature
cosign verify ghcr.io/<username>/hcp-checkmarx-listener:v1.0.0 \
  --certificate-identity-regexp=".*" \
  --certificate-oidc-issuer-regexp=".*"

# Verify OpenShift image signature  
cosign verify quay.io/<username>/hcp-checkmarx-listener-openshift:v1.0.0 \
  --certificate-identity-regexp=".*" \
  --certificate-oidc-issuer-regexp=".*"
```

## Monitoring and Troubleshooting

### Build Status
- Check Actions tab for build status
- Review build logs for any failures
- Security scan results appear in Security tab

### Image Security
- View vulnerability reports in Security → Code scanning
- Monitor SBOM artifacts for dependency tracking
- Check container signatures for supply chain verification

### Release Process
- Automated releases create GitHub releases
- Binary artifacts available for download
- Container images tagged with semantic versions

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Docker Build  │    │ OpenShift Build │    │Security/Release │
│                 │    │                 │    │                 │
│ • Multi-arch    │    │ • UBI base      │    │ • Vuln scanning │
│ • GHCR push     │    │ • Quay.io push  │    │ • Image signing │
│ • Trivy scan    │    │ • OCP labels    │    │ • SLSA attest   │
│ • SBOM gen      │    │ • K8s manifests │    │ • Releases      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │   Registries    │
                    │                 │
                    │ • ghcr.io       │
                    │ • quay.io       │
                    │ • Signed images │
                    │ • SBOM metadata │
                    └─────────────────┘
```

## Best Practices

1. **Branch Protection:** Use branch protection rules on main branches
2. **Secret Management:** Store sensitive tokens in GitHub Secrets
3. **Registry Access:** Use minimal permissions for registry tokens  
4. **Image Scanning:** Review security scan results before deployment
5. **Version Tags:** Use semantic versioning for releases
6. **Image Verification:** Always verify signatures in production