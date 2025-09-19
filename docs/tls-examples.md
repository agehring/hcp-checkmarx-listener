# TLS Configuration Examples

## Self-Signed Certificate (Quick Start)
```bash
# Start with self-signed certificate
./hcp-checkmarx-listener --tls --tls-self-signed --listen-port 8443 --config config.json

# Using environment variables
export TLS_ENABLED=true
export TLS_SELF_SIGNED=true
./hcp-checkmarx-listener --listen-port 8443 --config config.json
```

## Existing Certificate Files
```bash
# Using existing certificate files
./hcp-checkmarx-listener --tls --tls-cert /path/to/cert.crt --tls-key /path/to/key.key --listen-port 443 --config config.json

# With certificate file generation for persistence
./hcp-checkmarx-listener --tls --tls-self-signed --tls-cert ./server.crt --tls-key ./server.key --listen-port 8443 --config config.json
```

## Docker/Container TLS
```bash
# Mount certificate volume and run with TLS
docker run -p 443:443 -v /host/certs:/app/certs hcp-checkmarx-listener --tls --tls-cert /app/certs/server.crt --tls-key /app/certs/server.key --listen-port 443

# Self-signed in container
docker run -p 8443:8443 hcp-checkmarx-listener --tls --tls-self-signed --listen-port 8443
```

## Testing TLS Connection
```bash
# Test HTTPS connection
curl -k https://localhost:8443/health

# View certificate details
openssl s_client -connect localhost:8443 -servername localhost
```