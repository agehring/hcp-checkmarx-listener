# Production Logging Configuration Summary

This document outlines the comprehensive logging improvements implemented for the HCP Terraform-Checkmarx One integration service.

## Enhanced Logging Features

### 1. Server Startup Logging
**What's Logged:** Server startup with clear protocol and port information
**Format:** 
```
Server: Starting HTTP server on :8080
Server: Starting HTTPS server on :8443 (TLS enabled)
```
**When:** Always logged during server startup

### 2. Checkmarx One Authentication Logging
**What's Logged:** Authentication success for both API Key and OAuth2 methods
**Format:**
```
Auth: Checkmarx One API Key authentication successful
Auth: Checkmarx One OAuth2 authentication successful (tenant: your-tenant)
```
**When:** Always logged on successful authentication (failures are already logged as errors)

### 3. Connection Source Logging
**What's Logged:** Source IP for supported endpoints only
**Format:**
```
Connection: GET /health from 192.168.1.100:54321
Connection: POST /api/run-task from 10.0.0.5:43210
```
**When:** Every request to `/api/run-task` and `/health` endpoints (ignores 418 teapot responses)

### 4. HMAC Security Logging
**What's Logged:** HMAC validation results for security monitoring
**Format:**
```
Security: HMAC validation successful for pre_apply stage from 10.0.0.5:43210
Security: HMAC validation failed - missing signature from 192.168.1.100:54321
Security: HMAC validation failed - invalid signature from 10.0.0.5:43210
Security: HMAC validation disabled - allowing request from 10.0.0.5:43210
```
**When:** Always logged for HMAC validation attempts on actionable stages

### 5. Scan Results Logging
**What's Logged:** Total findings count and severity breakdown
**Format:**
```
Scan: Retrieved 42 IaC findings from Checkmarx One (scan ID: 12345678-1234-5678-9abc-123456789012)
Scan: Severity breakdown - map[CRITICAL:3 HIGH:8 MEDIUM:15 LOW:16]
```
**When:** Always logged when scan results are retrieved

### 6. Policy Evaluation Logging
**What's Logged:** Policy check initiation and results
**Format:**
```
Policy: Checking policy violations for scan ID: 12345678-1234-5678-9abc-123456789012
Policy: Check completed - Status: PASSED, BreakBuild: false, Policies evaluated: 5
```
**When:** Always logged during policy evaluation

## Log Message Prefixes

All production log messages use clear prefixes for easy filtering and monitoring:

- `Server:` - Server startup and configuration
- `Auth:` - Authentication events
- `Connection:` - Endpoint access logging
- `Security:` - HMAC validation and security events
- `Scan:` - Checkmarx scan results and findings
- `Policy:` - Policy evaluation and violation checking

## Debug Mode

When debug mode is enabled (`"debug": true` in config.json), additional verbose logging is available:
- Detailed request/response bodies
- Header information (with sensitive data masked)
- Step-by-step processing details
- Raw API responses and debugging information

## Log Level Recommendations

### Production Environment
- Standard logging provides sufficient operational visibility
- Monitor `Security:` messages for authentication issues
- Track `Scan:` and `Policy:` messages for scan results
- Alert on ERROR level messages

### Development/Testing Environment
- Enable debug mode for detailed troubleshooting
- Use debug logging to trace request flow
- Verify HMAC signatures and payload processing

## Example Production Log Output

```
2025/09/18 16:30:15 Auth: Checkmarx One OAuth2 authentication successful (tenant: acme-corp)
2025/09/18 16:30:15 Server: Starting HTTPS server on :8443 (TLS enabled)
2025/09/18 16:30:22 Connection: POST /api/run-task from 10.0.0.5:43210
2025/09/18 16:30:22 Security: HMAC validation successful for pre_apply stage from 10.0.0.5:43210
2025/09/18 16:30:25 Scan: Retrieved 42 IaC findings from Checkmarx One (scan ID: 12345678-1234-5678-9abc-123456789012)
2025/09/18 16:30:25 Scan: Severity breakdown - map[CRITICAL:3 HIGH:8 MEDIUM:15 LOW:16]
2025/09/18 16:30:26 Policy: Checking policy violations for scan ID: 12345678-1234-5678-9abc-123456789012
2025/09/18 16:30:26 Policy: Check completed - Status: FAILED, BreakBuild: true, Policies evaluated: 5
```

## Benefits

1. **Operational Visibility:** Clear insight into service operation without debug overhead
2. **Security Monitoring:** Track authentication and HMAC validation events
3. **Performance Tracking:** Monitor scan processing and results
4. **Troubleshooting:** Structured logging makes issue diagnosis easier
5. **Compliance:** Security event logging for audit requirements