# Certificate-Based Authentication for TimescaleDB

This document explains how to set up certificate-based authentication (mTLS) between the Anker Solix Exporter and TimescaleDB.

## Overview

Certificate-based authentication provides enhanced security by using X.509 certificates for mutual TLS authentication. This eliminates the need to transmit passwords over the network and provides stronger authentication guarantees.

## Prerequisites

- TimescaleDB/PostgreSQL server configured to accept certificate authentication
- Certificate Authority (CA) for signing certificates
- Client certificate and key for the exporter

## Certificate Generation

### 1. Generate CA Certificate (if you don't have one)

```bash
# Generate CA private key
openssl genrsa -out ca-key.pem 4096

# Generate CA certificate
openssl req -new -x509 -days 3650 -key ca-key.pem -out ca-cert.pem \
  -subj "/CN=TimescaleDB-CA"
```

### 2. Generate Client Certificate for Exporter

```bash
# Generate client private key
openssl genrsa -out client-key.pem 4096

# Generate certificate signing request
openssl req -new -key client-key.pem -out client-req.csr \
  -subj "/CN=anker_exporter"

# Sign the client certificate with CA
openssl x509 -req -days 365 -in client-req.csr \
  -CA ca-cert.pem -CAkey ca-key.pem -CAcreateserial \
  -out client-cert.pem

# Remove the CSR (no longer needed)
rm client-req.csr
```

### 3. Generate Server Certificate for TimescaleDB

```bash
# Generate server private key
openssl genrsa -out server-key.pem 4096

# Generate certificate signing request
openssl req -new -key server-key.pem -out server-req.csr \
  -subj "/CN=timescaledb"

# Sign the server certificate with CA
openssl x509 -req -days 365 -in server-req.csr \
  -CA ca-cert.pem -CAkey ca-key.pem -CAcreateserial \
  -out server-cert.pem

# Remove the CSR
rm server-req.csr

# Set proper permissions for PostgreSQL
chmod 600 server-key.pem
```

## TimescaleDB Configuration

### 1. Copy Certificates to Database Server

```bash
# Copy server certificates to PostgreSQL data directory
cp server-cert.pem /var/lib/postgresql/data/
cp server-key.pem /var/lib/postgresql/data/
cp ca-cert.pem /var/lib/postgresql/data/
chown postgres:postgres /var/lib/postgresql/data/*.pem
chmod 600 /var/lib/postgresql/data/server-key.pem
chmod 644 /var/lib/postgresql/data/server-cert.pem
chmod 644 /var/lib/postgresql/data/ca-cert.pem
```

### 2. Update postgresql.conf

Add or modify these settings:

```ini
ssl = on
ssl_cert_file = 'server-cert.pem'
ssl_key_file = 'server-key.pem'
ssl_ca_file = 'ca-cert.pem'
ssl_prefer_server_ciphers = on
ssl_ciphers = 'HIGH:MEDIUM:+3DES:!aNULL'
```

### 3. Update pg_hba.conf

Replace password authentication with certificate authentication:

```
# TYPE  DATABASE        USER            ADDRESS                 METHOD
hostssl anker_solix     anker_exporter  0.0.0.0/0              cert clientcert=verify-full
hostssl anker_solix     anker_exporter  ::/0                   cert clientcert=verify-full
```

### 4. Create Database User Mapped to Certificate

```sql
-- Create user (if not exists)
CREATE USER anker_exporter;

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE anker_solix TO anker_exporter;

-- Create user mapping for certificate authentication
-- The CN in the client certificate must match the username
```

### 5. Restart PostgreSQL

```bash
systemctl restart postgresql
# or for Docker
docker restart timescaledb
```

## Exporter Configuration

### Option 1: Using Configuration File

Update `config.yaml`:

```yaml
database:
  host: "timescaledb"
  port: 5432
  user: "anker_exporter"
  password: ""  # Not needed with cert auth, but required field
  database: "anker_solix"
  sslmode: "verify-full"  # or "verify-ca" or "require"
  migrations_path: "/etc/anker-solix-exporter/migrations"
  
  # Certificate paths
  sslcert: "/etc/database-certs/client-cert.pem"
  sslkey: "/etc/database-certs/client-key.pem"
  sslrootcert: "/etc/database-certs/ca-cert.pem"
```

### Option 2: Using Environment Variables

```bash
export DB_HOST=timescaledb
export DB_PORT=5432
export DB_USER=anker_exporter
export DB_PASSWORD=dummy  # Still required but not used
export DB_NAME=anker_solix
export DB_SSLMODE=verify-full
export DB_SSLCERT=/etc/database-certs/client-cert.pem
export DB_SSLKEY=/etc/database-certs/client-key.pem
export DB_SSLROOTCERT=/etc/database-certs/ca-cert.pem
```

## Docker Deployment

### docker-compose.yml

```yaml
services:
  anker-solix-exporter:
    # ... other configuration ...
    environment:
      - DB_SSLMODE=verify-full
      - DB_SSLCERT=/certs/client-cert.pem
      - DB_SSLKEY=/certs/client-key.pem
      - DB_SSLROOTCERT=/certs/ca-cert.pem
    volumes:
      - ./certs:/certs:ro
```

## Kubernetes/Helm Deployment

### 1. Create Secret with Certificates

```bash
kubectl create secret generic database-tls-certs \
  --from-file=tls.crt=client-cert.pem \
  --from-file=tls.key=client-key.pem \
  --from-file=ca.crt=ca-cert.pem \
  -n your-namespace
```

### 2. Update values.yaml

```yaml
database:
  host: "timescaledb.monitoring.svc.cluster.local"
  port: 5432
  user: "anker_exporter"
  password: "dummy"  # Not used but required
  name: "anker_solix"
  sslmode: "verify-full"
  
  tls:
    enabled: true
    secretName: "database-tls-certs"
```

### 3. Deploy

```bash
helm upgrade --install anker-solix-exporter ./deployments/helm/anker-solix-exporter \
  -f values-production.yaml
```

## SSL Modes Explained

- **disable**: No SSL (not recommended for production)
- **require**: Use SSL but don't verify the server certificate
- **verify-ca**: Use SSL and verify the server certificate is signed by a trusted CA
- **verify-full**: Use SSL, verify the certificate, and verify the hostname matches

For production with certificates, use `verify-full` for maximum security.

## Testing the Connection

### Test with psql

```bash
psql "host=timescaledb port=5432 dbname=anker_solix user=anker_exporter \
  sslmode=verify-full \
  sslcert=client-cert.pem \
  sslkey=client-key.pem \
  sslrootcert=ca-cert.pem"
```

### Check SSL Connection in PostgreSQL

```sql
SELECT * FROM pg_stat_ssl WHERE pid = pg_backend_pid();
```

## Troubleshooting

### "certificate verify failed"

- Ensure the CA certificate is correctly specified
- Check that server certificate is signed by the CA
- Verify certificate hasn't expired

### "certificate authentication failed for user"

- Ensure the CN in the client certificate matches the database username
- Check pg_hba.conf is configured for cert authentication
- Verify certificate permissions (key should be mode 600)

### "SSL connection has been closed unexpectedly"

- Check PostgreSQL logs for details
- Ensure server certificate and key are readable by PostgreSQL
- Verify ssl is enabled in postgresql.conf

## Security Best Practices

1. **Protect Private Keys**: Keep key files with 600 permissions, owned by the appropriate user
2. **Rotate Certificates**: Set reasonable expiration dates and rotate before expiry
3. **Use Strong Ciphers**: Configure PostgreSQL to use strong cipher suites
4. **Restrict Access**: Use firewall rules in addition to certificate auth
5. **Monitor Certificates**: Set up alerts for certificate expiration
6. **Secure Storage**: Store certificates in secrets management systems (Vault, K8s Secrets, etc.)

## Certificate Expiration Monitoring

Add monitoring for certificate expiration:

```bash
# Check certificate expiration
openssl x509 -in client-cert.pem -noout -enddate

# Get days until expiration
openssl x509 -in client-cert.pem -noout -checkend $((86400 * 30))
```

## References

- [PostgreSQL SSL Support](https://www.postgresql.org/docs/current/ssl-tcp.html)
- [PostgreSQL Certificate Authentication](https://www.postgresql.org/docs/current/auth-cert.html)
- [OpenSSL Certificate Commands](https://www.openssl.org/docs/man1.1.1/man1/)
