# Container Registry Setup Guide

This guide covers setting up container registries for the HCP Checkmarx Listener on OpenShift.

## OpenShift Internal Registry

### Enable Internal Registry

```bash
# Check if internal registry is enabled
oc get configs.imageregistry.operator.openshift.io cluster

# Enable if needed
oc patch configs.imageregistry.operator.openshift.io cluster --type merge --patch '{"spec":{"managementState":"Managed"}}'

# Expose registry route
oc patch configs.imageregistry.operator.openshift.io/cluster --patch '{"spec":{"defaultRoute":true}}' --type=merge
```

### Build and Push to Internal Registry

```bash
# Get registry URL
REGISTRY_URL=$(oc get route default-route -n openshift-image-registry --template='{{ .spec.host }}')

# Login to registry
oc registry login

# Build and push
./build-container.sh \
  --registry $REGISTRY_URL \
  --namespace myproject \
  --tag latest \
  --push
```

## Quay.io Registry

### Setup

1. Create account at https://quay.io
2. Create organization and repository
3. Generate robot account with push/pull permissions

### Authentication

```bash
# Create pull secret
oc create secret docker-registry quay-pull-secret \
  --docker-server=quay.io \
  --docker-username=your-org+robot \
  --docker-password=robot-token \
  --docker-email=your-email@example.com \
  -n hcp-checkmarx

# Link to service account
oc patch serviceaccount hcp-checkmarx-listener \
  -p '{"imagePullSecrets": [{"name": "quay-pull-secret"}]}' \
  -n hcp-checkmarx
```

### Build and Push

```bash
# Login
docker login quay.io

# Build and push
./build-container.sh \
  --registry quay.io \
  --namespace your-org \
  --tag v1.0.0 \
  --push
```

## Red Hat Registry (registry.redhat.io)

### Authentication

```bash
# Create pull secret with Red Hat credentials
oc create secret docker-registry redhat-pull-secret \
  --docker-server=registry.redhat.io \
  --docker-username=your-redhat-username \
  --docker-password=your-redhat-password \
  --docker-email=your-email@example.com \
  -n hcp-checkmarx
```

## Private Registry

### Self-signed Certificates

```bash
# Add registry to trusted
oc create configmap registry-ca \
  --from-file=ca.crt=/path/to/registry-ca.crt \
  -n openshift-config

oc patch image.config.openshift.io/cluster \
  --patch '{"spec":{"additionalTrustedCA":{"name":"registry-ca"}}}' \
  --type=merge
```

### Insecure Registry (Not Recommended)

```bash
oc patch image.config.openshift.io/cluster \
  --patch '{"spec":{"registrySources":{"insecureRegistries":["your-registry.com"]}}}' \
  --type=merge
```

## Multi-Architecture Builds

### Using Buildah for Multi-arch

```bash
# Build for multiple architectures
./build-container.sh --buildah --tag multi-arch

# Create manifest list
buildah manifest create hcp-checkmarx-listener:multi-arch

# Add architectures
buildah manifest add hcp-checkmarx-listener:multi-arch hcp-checkmarx-listener:amd64
buildah manifest add hcp-checkmarx-listener:multi-arch hcp-checkmarx-listener:arm64

# Push manifest
buildah manifest push --all hcp-checkmarx-listener:multi-arch docker://quay.io/your-org/hcp-checkmarx-listener:multi-arch
```

## Security Scanning

### Red Hat Advanced Cluster Security

```yaml
apiVersion: platform.stackrox.io/v1alpha1
kind: SecuredCluster
metadata:
  name: stackrox-secured-cluster-services
spec:
  scannerComponent:
    scannerAnalyzerComponent:
      enabled: true
      replicas: 1
```

### Clair Scanning

```bash
# Install Clair operator
oc apply -f https://operatorhub.io/install/clair.yaml

# Create ClairApp instance
oc apply -f - <<EOF
apiVersion: clair.coreos.com/v1alpha1
kind: ClairApp
metadata:
  name: clair-scanner
spec:
  database:
    type: postgres
EOF
```

## Registry Cleanup

### Prune Old Images

```bash
# Prune images older than 30 days
oc adm prune images --keep-tag-revisions=3 --keep-younger-than=720h --confirm

# Prune unused images
oc adm prune images --prune-over-size-limit --confirm
```

### Automated Cleanup

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: registry-cleanup
spec:
  schedule: "0 2 * * 0"  # Weekly at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: prune
            image: registry.redhat.io/openshift4/ose-cli:latest
            command: ["oc", "adm", "prune", "images", "--keep-tag-revisions=5", "--confirm"]
          restartPolicy: OnFailure
```