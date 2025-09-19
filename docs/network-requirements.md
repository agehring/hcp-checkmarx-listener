# Network Requirements and Configuration Guide

This document provides detailed network requirements, configuration examples, and troubleshooting guidance for deploying the HCP Terraform Checkmarx One integration in enterprise environments.

## Table of Contents

1. [Network Architecture Overview](#network-architecture-overview)
2. [Regional Endpoints](#regional-endpoints)
3. [Firewall Configuration](#firewall-configuration)
4. [Proxy Configuration](#proxy-configuration)
5. [DNS Requirements](#dns-requirements)
6. [Security Considerations](#security-considerations)
7. [Troubleshooting](#troubleshooting)
8. [Monitoring and Logging](#monitoring-and-logging)

## Network Architecture Overview

### High-Level Architecture

```
Internet/Cloud                 Enterprise Network                    Internal Network
┌─────────────────┐           ┌─────────────────┐                  ┌─────────────────┐
│                 │           │                 │                  │                 │
│ HCP Terraform   │  HTTPS    │   Web Proxy/    │      HTTP(S)     │ Listener Service│
│app.terraform.io │ ────────→ │    Firewall     │ ───────────────→ │   Container     │
│                 │   :443    │                 │      :8080       │                 │
└─────────────────┘           └─────────────────┘                  └─────────────────┘
                                        │                                    │
┌─────────────────┐                     │                                    │
│                 │                     │                                    │
│ Checkmarx One   │  HTTPS              │      HTTPS                         │
│*.checkmarx.net  │ ←───────────────────┼────────────────────────────────────┘
│                 │   :443              │      :443
└─────────────────┘                     │
                                        │
┌─────────────────┐                     │
│                 │                     │
│   AWS S3        │  HTTPS              │
│ File Storage    │ ←───────────────────┘
│                 │   :443
└─────────────────┘
```

### Data Flow Sequence

1. **Webhook Reception**: HCP Terraform sends run task webhook to listener service
2. **Authentication**: Listener authenticates with Checkmarx One
3. **Configuration Download**: Listener downloads Terraform workspace from HCP Terraform
4. **File Upload**: Listener uploads workspace archive to Checkmarx storage (S3)
5. **Scan Submission**: Listener submits IaC scan job to Checkmarx One
6. **Status Polling**: Listener polls Checkmarx for scan completion
7. **Results Retrieval**: Listener fetches detailed scan results
8. **Callback**: Listener sends results back to HCP Terraform

## Regional Endpoints

### Checkmarx One Regions

| Region | API Endpoint | Auth Endpoint | S3 Region | Geographic Location |
|--------|--------------|---------------|-----------|-------------------|
| US | `https://ast.checkmarx.net` | `https://iam.checkmarx.net` | us-east-1 | United States East |
| US2 | `https://us.ast.checkmarx.net` | `https://us.iam.checkmarx.net` | us-west-2 | United States West |
| EU | `https://eu.ast.checkmarx.net` | `https://eu.iam.checkmarx.net` | eu-west-1 | Europe West |
| EU2 | `https://eu-2.ast.checkmarx.net` | `https://eu-2.iam.checkmarx.net` | eu-central-1 | Europe Central |
| DEU | `https://deu.ast.checkmarx.net` | `https://deu.iam.checkmarx.net` | eu-central-1 | Germany |
| ANZ | `https://anz.ast.checkmarx.net` | `https://anz.iam.checkmarx.net` | ap-southeast-2 | Australia/New Zealand |
| IND | `https://ind.ast.checkmarx.net` | `https://ind.iam.checkmarx.net` | ap-south-1 | India |
| SNG | `https://sng.ast.checkmarx.net` | `https://sng.iam.checkmarx.net` | ap-southeast-1 | Singapore |
| MEA | `https://mea.ast.checkmarx.net` | `https://mea.iam.checkmarx.net` | me-south-1 | Middle East/Africa |

### IP Address Ranges

For environments requiring IP whitelisting, consult:
- **HCP Terraform**: [IP Ranges Documentation](https://www.terraform.io/docs/cloud/architectural-details/ip-ranges.html)
- **Checkmarx One**: Contact Checkmarx support for current IP ranges
- **AWS S3**: [AWS IP Ranges](https://docs.aws.amazon.com/general/latest/gr/aws-ip-ranges.html)

## Firewall Configuration

### Outbound Rules (Required)

#### Basic Connectivity
```bash
# HCP Terraform API
ALLOW TCP FROM listener-subnet TO app.terraform.io:443
ALLOW TCP FROM listener-subnet TO registry.terraform.io:443

# Checkmarx One APIs (choose your region)
ALLOW TCP FROM listener-subnet TO ast.checkmarx.net:443
ALLOW TCP FROM listener-subnet TO iam.checkmarx.net:443

# AWS S3 for file uploads (region-specific)
ALLOW TCP FROM listener-subnet TO *.s3.us-east-1.amazonaws.com:443
ALLOW TCP FROM listener-subnet TO *.s3.amazonaws.com:443
```

#### Multi-Region Support
```bash
# All Checkmarx regions (if multi-region deployment)
ALLOW TCP FROM listener-subnet TO *.ast.checkmarx.net:443
ALLOW TCP FROM listener-subnet TO *.iam.checkmarx.net:443

# All AWS S3 regions
ALLOW TCP FROM listener-subnet TO *.s3.*.amazonaws.com:443
```

#### DNS Resolution
```bash
ALLOW UDP FROM listener-subnet TO dns-servers:53
ALLOW TCP FROM listener-subnet TO dns-servers:53
```

### Inbound Rules (Required)

```bash
# HCP Terraform webhooks
ALLOW TCP FROM app.terraform.io TO listener-service:8080

# Health checks (optional)
ALLOW TCP FROM load-balancer TO listener-service:8080
ALLOW TCP FROM monitoring-subnet TO listener-service:8080
```

### Firewall Rule Templates

#### pfSense/OPNsense
```xml
<rule>
    <type>pass</type>
    <interface>lan</interface>
    <source>
        <address>listener-subnet</address>
    </source>
    <destination>
        <address>ast.checkmarx.net</address>
        <port>443</port>
    </destination>
    <protocol>tcp</protocol>
    <description>Checkmarx One API Access</description>
</rule>
```

#### iptables
```bash
# Outbound to Checkmarx
iptables -A OUTPUT -s listener-subnet -d ast.checkmarx.net -p tcp --dport 443 -j ACCEPT

# Inbound from HCP Terraform
iptables -A INPUT -s app.terraform.io -d listener-service -p tcp --dport 8080 -j ACCEPT
```

#### Windows Firewall
```powershell
# Outbound rule for Checkmarx
New-NetFirewallRule -DisplayName "Checkmarx One API" -Direction Outbound -Protocol TCP -RemoteAddress "ast.checkmarx.net" -RemotePort 443 -Action Allow

# Inbound rule for HCP Terraform
New-NetFirewallRule -DisplayName "HCP Terraform Webhook" -Direction Inbound -Protocol TCP -LocalPort 8080 -RemoteAddress "app.terraform.io" -Action Allow
```

## Proxy Configuration

### HTTP/HTTPS Proxy Setup

#### Environment Variables
```bash
export HTTPS_PROXY=https://proxy.company.com:8080
export HTTP_PROXY=http://proxy.company.com:8080
export NO_PROXY=localhost,127.0.0.1,*.company.local
```

#### Configuration File
```yaml
# config.yaml
proxy:
  url: "https://proxy.company.com:8080"
  username: "proxy-user"
  password: "proxy-password"
  no_proxy:
    - "localhost"
    - "127.0.0.1"
    - "*.internal.company.com"
    - "10.0.0.0/8"
    - "172.16.0.0/12"
    - "192.168.0.0/16"
```

### Corporate Proxy Authentication

#### Basic Authentication
```yaml
proxy:
  url: "http://proxy.company.com:8080"
  auth:
    type: "basic"
    username: "domain\\username"
    password: "password"
```

#### NTLM Authentication
```yaml
proxy:
  url: "http://proxy.company.com:8080"
  auth:
    type: "ntlm"
    domain: "COMPANY"
    username: "username"
    password: "password"
```

#### Certificate Authentication
```yaml
proxy:
  url: "https://proxy.company.com:8080"
  auth:
    type: "certificate"
    cert_file: "/etc/ssl/certs/client.crt"
    key_file: "/etc/ssl/private/client.key"
    ca_file: "/etc/ssl/certs/ca.crt"
```

### Proxy Configuration Testing

```bash
# Test proxy connectivity
curl -v --proxy http://proxy.company.com:8080 https://ast.checkmarx.net/api/health

# Test authentication
curl -v --proxy-user username:password --proxy http://proxy.company.com:8080 https://app.terraform.io
```

## DNS Requirements

### Required DNS Records

#### Forward Lookup
```dns
# HCP Terraform
app.terraform.io.               A       52.86.200.106
app.terraform.io.               A       34.205.106.222

# Checkmarx One (US region example)
ast.checkmarx.net.              CNAME   checkmarx-prod-us.herokuapp.com.
iam.checkmarx.net.              CNAME   auth-prod-us.herokuapp.com.

# AWS S3 (regional)
*.s3.us-east-1.amazonaws.com.   CNAME   s3.us-east-1.amazonaws.com.
s3.us-east-1.amazonaws.com.     A       52.216.200.0/24
```

#### Corporate DNS Configuration
```bind
# /etc/bind/zones/company.local
$TTL 3600
@   IN  SOA ns1.company.local. admin.company.local. (
    2024091901  ; Serial
    3600        ; Refresh
    1800        ; Retry
    604800      ; Expire
    86400       ; Minimum TTL
)

; Forward zones for external services
ast.checkmarx.net.      300 IN  A       1.2.3.4     ; Proxy IP
iam.checkmarx.net.      300 IN  A       1.2.3.4     ; Proxy IP
app.terraform.io.       300 IN  A       1.2.3.4     ; Proxy IP
```

### DNS Resolution Testing

```bash
# Test DNS resolution
nslookup ast.checkmarx.net
dig +short app.terraform.io
host iam.checkmarx.net

# Test from container
kubectl exec -it listener-pod -- nslookup ast.checkmarx.net
```

## Security Considerations

### TLS/SSL Configuration

#### Certificate Requirements
```yaml
# Minimum TLS configuration
tls:
  min_version: "1.2"
  cipher_suites:
    - "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
    - "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
    - "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384"
    - "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"
  
  # Certificate validation
  verify_certificates: true
  ca_bundle: "/etc/ssl/certs/ca-certificates.crt"
```

#### Certificate Pinning (Optional)
```yaml
# Pin critical service certificates
certificate_pins:
  "ast.checkmarx.net":
    - "sha256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
  "app.terraform.io":
    - "sha256:BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB="
```

### Network Segmentation

#### Recommended VLAN Structure
```
VLAN 100: DMZ/Internet-facing services
VLAN 200: Application services (Listener)
VLAN 300: Database/Storage services
VLAN 400: Management/Monitoring
```

#### Security Groups (AWS/Cloud)
```json
{
  "SecurityGroups": [
    {
      "GroupName": "hcp-checkmarx-listener-sg",
      "Rules": [
        {
          "Type": "ingress",
          "Protocol": "tcp",
          "Port": 8080,
          "Source": "app.terraform.io/32"
        },
        {
          "Type": "egress",
          "Protocol": "tcp",
          "Port": 443,
          "Destination": "ast.checkmarx.net/32"
        }
      ]
    }
  ]
}
```

## Troubleshooting

### Network Connectivity Issues

#### Test Basic Connectivity
```bash
# Test outbound HTTPS connectivity
curl -v https://ast.checkmarx.net/api/health
curl -v https://app.terraform.io

# Test DNS resolution
nslookup ast.checkmarx.net
dig app.terraform.io

# Test proxy connectivity
curl -v --proxy http://proxy:8080 https://ast.checkmarx.net
```

#### Common Issues and Solutions

| Issue | Symptoms | Solution |
|-------|----------|----------|
| DNS Resolution Failure | "no such host" errors | Configure corporate DNS or add hosts entries |
| Proxy Authentication | 407 Proxy Auth Required | Verify proxy credentials and auth method |
| Firewall Blocking | Connection timeout | Review firewall rules and logs |
| Certificate Issues | TLS handshake failures | Update CA bundle or disable verification (not recommended) |
| Regional Endpoint | Wrong region errors | Verify Checkmarx region configuration |

#### Debug Commands
```bash
# Network trace (Linux)
tcpdump -i any host ast.checkmarx.net
ss -tuln | grep :8080

# Windows network trace
netsh trace start capture=yes
netstat -an | findstr :8080

# Container networking (Kubernetes)
kubectl exec -it listener-pod -- netstat -tuln
kubectl logs listener-pod --follow
```

### Service Logs Analysis

#### Enable Debug Logging
```yaml
# config.yaml
debug: true
logging:
  level: "debug"
  format: "json"
  
# Environment variable
DEBUG=true
```

#### Log Patterns to Monitor
```bash
# Successful flows
grep "Auth: Checkmarx One.*authentication successful" /var/log/listener.log
grep "Connection: Received webhook from.*app.terraform.io" /var/log/listener.log

# Error patterns
grep "ERROR" /var/log/listener.log | grep -E "(timeout|connection|auth)"
grep "Security: HMAC validation failed" /var/log/listener.log
```

## Monitoring and Logging

### Network Monitoring

#### Metrics to Track
```prometheus
# Connection success rates
http_requests_total{endpoint="checkmarx_api"}
http_request_duration_seconds{endpoint="checkmarx_api"}

# Error rates
http_requests_total{status=~"4..|5.."}

# DNS resolution time
dns_resolution_duration_seconds
```

#### Health Check Endpoints
```bash
# Service health
curl http://localhost:8080/health

# Readiness check
curl http://localhost:8080/ready

# Metrics
curl http://localhost:8080/metrics
```

### Log Aggregation

#### Structured Logging Example
```json
{
  "timestamp": "2024-09-19T10:30:00Z",
  "level": "info",
  "component": "network",
  "message": "Successful connection to Checkmarx One",
  "details": {
    "endpoint": "ast.checkmarx.net",
    "region": "US",
    "response_time_ms": 245,
    "status_code": 200
  }
}
```

#### Log Shipping Configuration
```yaml
# Fluentd/Fluent Bit
<source>
  @type tail
  path /var/log/listener.log
  pos_file /var/log/fluentd/listener.log.pos
  tag hcp.listener
  format json
</source>

<match hcp.listener>
  @type elasticsearch
  host elasticsearch.company.com
  port 9200
  index_name hcp-listener-logs
</match>
```

This comprehensive network documentation should help enterprise teams properly configure, secure, and troubleshoot the HCP Terraform Checkmarx One integration in their environments.