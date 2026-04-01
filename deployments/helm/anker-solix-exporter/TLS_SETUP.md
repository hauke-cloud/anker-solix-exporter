# Database TLS Configuration

This Helm chart supports optional certificate-based authentication for the TimescaleDB connection.

## Quick Start with TLS

### 1. Create a Kubernetes Secret with Certificates

```bash
kubectl create secret generic database-tls-certs \
  --from-file=tls.crt=path/to/client-cert.pem \
  --from-file=tls.key=path/to/client-key.pem \
  --from-file=ca.crt=path/to/ca-cert.pem \
  -n your-namespace
```

### 2. Enable TLS in values.yaml

```yaml
database:
  sslmode: "verify-full"
  tls:
    enabled: true
    secretName: "database-tls-certs"
```

### 3. Deploy

```bash
helm upgrade --install anker-solix-exporter . -f values.yaml
```

## Alternative: Inline Certificate Content

For development/testing only (not recommended for production):

```yaml
database:
  tls:
    enabled: true
    cert: |
      -----BEGIN CERTIFICATE-----
      ...
      -----END CERTIFICATE-----
    key: |
      -----BEGIN PRIVATE KEY-----
      ...
      -----END PRIVATE KEY-----
    caCert: |
      -----BEGIN CERTIFICATE-----
      ...
      -----END CERTIFICATE-----
```

## Certificate Requirements

The secret must contain these keys:
- `tls.crt`: Client certificate (PEM format)
- `tls.key`: Client private key (PEM format)
- `ca.crt`: CA certificate (PEM format)

## SSL Modes

- `disable`: No SSL
- `require`: SSL required but no verification
- `verify-ca`: Verify server certificate
- `verify-full`: Verify certificate and hostname (recommended)

For detailed setup instructions, see [../../docs/CERTIFICATE_AUTH.md](../../docs/CERTIFICATE_AUTH.md).
