# Chainguard Container Images

This project uses [Chainguard Images](https://www.chainguard.dev/chainguard-images) for maximum security and minimal CVEs.

## Why Chainguard?

- **🛡️ Security First**: Images are rebuilt daily with latest security patches
- **📦 Minimal CVEs**: Designed specifically to minimize vulnerabilities
- **🔒 Provenance**: All images have verifiable supply chain security
- **⚡ Performance**: Smaller, faster images with reduced attack surface
- **🔄 Fresh Updates**: Continuous updates ensure latest security fixes

## Available Dockerfiles

### Production Dockerfile (Default)
- **File**: `Dockerfile`
- **Builder**: `cgr.dev/chainguard/go:latest-dev`
- **Runtime**: `cgr.dev/chainguard/static:latest`
- **Use Case**: Maximum security, static binary

### OpenShift Dockerfile (Workflow)
- **File**: Generated in `.github/workflows/openshift-build.yml`
- **Builder**: `cgr.dev/chainguard/go:latest-dev`
- **Runtime**: `cgr.dev/chainguard/glibc-dynamic:latest`
- **Use Case**: OpenShift compatibility with dynamic linking

### Alternative Options
- **Dockerfile.chainguard**: Pure Chainguard implementation for additional testing

## Chainguard Images Used

### Build Stage
```dockerfile
FROM cgr.dev/chainguard/go:latest-dev AS builder
```
- Contains Go toolchain and build dependencies
- Latest development tools for current Go version
- Minimal base with security patches
- No unnecessary packages or CVEs

### Runtime Stage (Static)
```dockerfile
FROM cgr.dev/chainguard/static:latest
```
- No shell, no package manager
- Only contains static libraries and certificates
- Maximum security for static binaries
- Continuously updated with security patches

### Runtime Stage (OpenShift)
```dockerfile
FROM cgr.dev/chainguard/glibc-dynamic:latest
```
- Compatible with OpenShift security constraints
- Includes glibc for dynamic linking
- Maintains minimal attack surface
- Regularly updated for security

## Security Benefits

1. **🔒 CVE-Free**: Near-zero CVEs in production images
2. **📋 SBOM Included**: Software Bill of Materials for transparency
3. **✅ Signed Images**: Cosign signatures for verification
4. **🛡️ Attestations**: Build provenance and security attestations
5. **🔄 Auto-Updates**: Daily rebuilds with latest patches

## Verification

Verify image signatures:
```bash
# Verify the Chainguard base image
cosign verify cgr.dev/chainguard/static:latest \
  --certificate-identity-regexp=".*@chainguard.dev" \
  --certificate-oidc-issuer=https://token.actions.githubusercontent.com

# Verify your built image (after build)
cosign verify ghcr.io/agehring/hcp-checkmarx-listener:latest \
  --certificate-identity-regexp=".*@users.noreply.github.com" \
  --certificate-oidc-issuer=https://token.actions.githubusercontent.com
```

## Migration Benefits

- **✅ ncurses**: Eliminated - not present in Chainguard images
- **✅ SQLite**: Eliminated - not present in Chainguard images  
- **✅ CVEs**: Minimized through continuous security updates
- **✅ Size**: Reduced container size and attack surface
- **✅ Performance**: Faster builds and deployments