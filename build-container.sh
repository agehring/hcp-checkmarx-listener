#!/bin/bash

# HCP Checkmarx Listener - Container Build Script for OpenShift/CRI-O
# This script builds and optionally pushes the container image

set -e

# Configuration
IMAGE_NAME="hcp-checkmarx-listener"
REGISTRY="${REGISTRY:-quay.io}"
NAMESPACE="${NAMESPACE:-your-org}"
TAG="${TAG:-latest}"
FULL_IMAGE="${REGISTRY}/${NAMESPACE}/${IMAGE_NAME}:${TAG}"

# Build context
BUILD_CONTEXT="$(dirname "$0")"
DOCKERFILE="${BUILD_CONTEXT}/Dockerfile"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Help function
show_help() {
    cat << EOF
Container Build Script for HCP Checkmarx Listener

Usage: $0 [OPTIONS]

OPTIONS:
    -h, --help          Show this help message
    -p, --push          Push image to registry after building
    -t, --tag TAG       Specify image tag (default: latest)
    -r, --registry URL  Specify registry URL (default: quay.io)
    -n, --namespace NS  Specify namespace/organization (default: your-org)
    --no-cache          Build without using cache
    --buildah           Use buildah instead of docker/podman
    --dry-run           Show commands that would be run without executing

ENVIRONMENT VARIABLES:
    REGISTRY            Container registry URL
    NAMESPACE           Registry namespace/organization
    TAG                 Image tag
    CONTAINER_TOOL      Container tool to use (docker, podman, buildah)

EXAMPLES:
    # Build with default settings
    $0

    # Build and push with custom tag
    $0 --push --tag v1.2.3

    # Build for OpenShift internal registry
    $0 --registry image-registry.openshift-image-registry.svc:5000 --namespace myproject

    # Build using buildah
    $0 --buildah

EOF
}

# Parse command line arguments
PUSH=false
NO_CACHE=false
USE_BUILDAH=false
DRY_RUN=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -p|--push)
            PUSH=true
            shift
            ;;
        -t|--tag)
            TAG="$2"
            FULL_IMAGE="${REGISTRY}/${NAMESPACE}/${IMAGE_NAME}:${TAG}"
            shift 2
            ;;
        -r|--registry)
            REGISTRY="$2"
            FULL_IMAGE="${REGISTRY}/${NAMESPACE}/${IMAGE_NAME}:${TAG}"
            shift 2
            ;;
        -n|--namespace)
            NAMESPACE="$2"
            FULL_IMAGE="${REGISTRY}/${NAMESPACE}/${IMAGE_NAME}:${TAG}"
            shift 2
            ;;
        --no-cache)
            NO_CACHE=true
            shift
            ;;
        --buildah)
            USE_BUILDAH=true
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Determine container tool
if [[ "$USE_BUILDAH" == "true" ]]; then
    CONTAINER_TOOL="buildah"
else
    CONTAINER_TOOL="${CONTAINER_TOOL:-docker}"
    # Check if podman is available and docker is not
    if ! command -v docker &> /dev/null && command -v podman &> /dev/null; then
        CONTAINER_TOOL="podman"
    fi
fi

# Verify container tool is available
if ! command -v "$CONTAINER_TOOL" &> /dev/null; then
    log_error "Container tool '$CONTAINER_TOOL' not found"
    exit 1
fi

# Build arguments
BUILD_ARGS=()
if [[ "$NO_CACHE" == "true" ]]; then
    if [[ "$CONTAINER_TOOL" == "buildah" ]]; then
        BUILD_ARGS+=(--no-cache)
    else
        BUILD_ARGS+=(--no-cache)
    fi
fi

# Add build labels
BUILD_LABELS=(
    "org.opencontainers.image.title=HCP Checkmarx Listener"
    "org.opencontainers.image.description=HCP Terraform integration with Checkmarx One for IaC scanning"
    "org.opencontainers.image.source=https://github.com/agehring/hcp-checkmarx-listener"
    "org.opencontainers.image.created=$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
    "org.opencontainers.image.version=${TAG}"
)

for label in "${BUILD_LABELS[@]}"; do
    BUILD_ARGS+=(--label "$label")
done

# Show build information
log_info "Building container image"
echo "  Image:      $FULL_IMAGE"
echo "  Tool:       $CONTAINER_TOOL"
echo "  Context:    $BUILD_CONTEXT"
echo "  Dockerfile: $DOCKERFILE"
echo "  Push:       $PUSH"

# Verify Dockerfile exists
if [[ ! -f "$DOCKERFILE" ]]; then
    log_error "Dockerfile not found: $DOCKERFILE"
    exit 1
fi

# Build command
if [[ "$CONTAINER_TOOL" == "buildah" ]]; then
    BUILD_CMD=(buildah build "${BUILD_ARGS[@]}" -f "$DOCKERFILE" -t "$FULL_IMAGE" "$BUILD_CONTEXT")
else
    BUILD_CMD=("$CONTAINER_TOOL" build "${BUILD_ARGS[@]}" -f "$DOCKERFILE" -t "$FULL_IMAGE" "$BUILD_CONTEXT")
fi

# Execute or show build command
if [[ "$DRY_RUN" == "true" ]]; then
    log_info "DRY RUN - Build command:"
    echo "  ${BUILD_CMD[*]}"
else
    log_info "Building image..."
    if "${BUILD_CMD[@]}"; then
        log_success "Image built successfully: $FULL_IMAGE"
    else
        log_error "Build failed"
        exit 1
    fi
fi

# Push if requested
if [[ "$PUSH" == "true" ]]; then
    PUSH_CMD=("$CONTAINER_TOOL" push "$FULL_IMAGE")
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "DRY RUN - Push command:"
        echo "  ${PUSH_CMD[*]}"
    else
        log_info "Pushing image to registry..."
        if "${PUSH_CMD[@]}"; then
            log_success "Image pushed successfully: $FULL_IMAGE"
        else
            log_error "Push failed"
            exit 1
        fi
    fi
fi

# Show next steps
if [[ "$DRY_RUN" != "true" ]]; then
    echo
    log_success "Build completed!"
    echo
    echo "Next steps:"
    echo "  1. Update image reference in k8s/deployment.yaml:"
    echo "     image: $FULL_IMAGE"
    echo
    echo "  2. Deploy to OpenShift:"
    echo "     oc apply -f k8s/"
    echo
    echo "  3. Or use the all-in-one manifest:"
    echo "     oc apply -f k8s/all-in-one.yaml"
    echo
    echo "  4. Set required secrets:"
    echo "     oc create secret generic hcp-checkmarx-secrets \\"
    echo "       --from-literal=checkmarx_api_key=\$(echo 'your-api-key' | base64) \\"
    echo "       --from-literal=hmac_secret=\$(echo 'your-hmac-secret' | base64)"
fi