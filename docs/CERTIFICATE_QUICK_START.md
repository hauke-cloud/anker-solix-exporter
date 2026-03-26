# Quick Reference: Certificate Authentication

## Enable Certificate Authentication

### Configuration File
```yaml
database:
  sslmode: "verify-full"
  sslcert: "/certs/client-cert.pem"
  sslkey: "/certs/client-key.pem"
  sslrootcert: "/certs/ca-cert.pem"
```

### Environment Variables
```bash
DB_SSLMODE=verify-full
DB_SSLCERT=/certs/client-cert.pem
DB_SSLKEY=/certs/client-key.pem
DB_SSLROOTCERT=/certs/ca-cert.pem
```

### Docker Compose
```yaml
services:
  anker-solix-exporter:
    environment:
      - DB_SSLMODE=verify-full
      - DB_SSLCERT=/certs/client-cert.pem
      - DB_SSLKEY=/certs/client-key.pem
      - DB_SSLROOTCERT=/certs/ca-cert.pem
    volumes:
      - ./certs:/certs:ro
```

### Kubernetes/Helm
```yaml
database:
  sslmode: "verify-full"
  tls:
    enabled: true
    secretName: "database-tls-certs"
```

Create secret:
```bash
kubectl create secret generic database-tls-certs \
  --from-file=tls.crt=client-cert.pem \
  --from-file=tls.key=client-key.pem \
  --from-file=ca.crt=ca-cert.pem
```

## SSL Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| `disable` | No SSL | Development only |
| `require` | SSL without verification | Basic encryption |
| `verify-ca` | Verify server certificate | Production with CA |
| `verify-full` | Full verification | **Recommended for production** |

## Certificate Requirements

- **Client Certificate** (`sslcert`): PEM format, CN must match database username
- **Client Key** (`sslkey`): PEM format, must match certificate
- **CA Certificate** (`sslrootcert`): PEM format, must have signed server cert

## File Permissions

```bash
chmod 600 client-key.pem    # Private key
chmod 644 client-cert.pem   # Client certificate
chmod 644 ca-cert.pem       # CA certificate
```

## Validation

The exporter validates:
- ✓ Certificate files exist
- ✓ Both cert and key provided together
- ✓ SSL mode not 'disable' when certs configured

## More Information

- [Full Setup Guide](CERTIFICATE_AUTH.md)
- [Helm TLS Setup](../deployments/helm/anker-solix-exporter/TLS_SETUP.md)
