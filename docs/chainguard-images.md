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
FROM cgr.dev/chainguard/go:1.23@sha256:4d4b56f8858db8a0aebf4fdb2db4f5e5dfa1194b486f1cf5a606c78ff2ad95f9 AS builder
```
- Contains Go 1.23 toolchain and build dependencies
- Pinned to specific digest for reproducible builds
- Minimal base with security patches
- No unnecessary packages or CVEs

### Runtime Stage (Static)
```dockerfile
FROM cgr.dev/chainguard/static:20241219@sha256:5ff428f8a48241da4e78d1c31f6e1f92d4c725f4e8ba7ff44ef64cecac6fa47
```
- No shell, no package manager
- Only contains static libraries and certificates
- Maximum security for static binaries
- Pinned to specific date tag and digest

### Runtime Stage (OpenShift)
```dockerfile
FROM cgr.dev/chainguard/glibc-dynamic:20241219@sha256:b81f9ca0853f5c67b2bab29b5d9e5e3de45e3a983ece956ba93e36e8be6ed1d7
```
- Compatible with OpenShift security constraints
- Includes glibc for dynamic linking
- Maintains minimal attack surface
- Pinned to specific date tag and digest

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