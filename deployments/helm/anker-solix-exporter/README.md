# Anker Solix Exporter Helm Chart

A Helm chart for deploying the Anker Solix solar system data exporter to Kubernetes.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- InfluxDB v2 instance
- Anker Solix account credentials

## Installation

### Quick Start

1. Add your credentials to a values file:

```bash
cat > my-values.yaml <<EOF
anker:
  email: "your.email@example.com"
  password: "your-password"
  country: "DE"
  pollInterval: "5m"

influxdb:
  url: "http://influxdb:8086"
  token: "your-influxdb-token"
  org: "my-org"
  bucket: "solar"

persistence:
  enabled: true
  size: 1Gi
EOF
```

2. Install the chart:

```bash
helm install anker-solix-exporter . \
  -f my-values.yaml \
  -n monitoring --create-namespace
```

### Using Existing Secrets

For production, it's recommended to create secrets separately:

```bash
# Create Anker credentials secret
kubectl create secret generic anker-credentials \
  --from-literal=ANKER_EMAIL=your.email@example.com \
  --from-literal=ANKER_PASSWORD=your-password \
  -n monitoring

# Create InfluxDB token secret
kubectl create secret generic influxdb-credentials \
  --from-literal=INFLUXDB_TOKEN=your-token \
  -n monitoring
```

Then reference them in values:

```yaml
anker:
  existingSecret: "anker-credentials"

influxdb:
  existingSecret: "influxdb-credentials"
```

## Configuration

See [values.yaml](values.yaml) for all available options.

### Key Configuration Options

| Parameter | Description | Default |
|-----------|-------------|---------|
| `anker.email` | Anker account email | `""` |
| `anker.password` | Anker account password | `""` |
| `anker.country` | Country code | `"DE"` |
| `anker.pollInterval` | Data polling interval | `"5m"` |
| `influxdb.url` | InfluxDB URL | `""` |
| `influxdb.token` | InfluxDB token | `""` |
| `influxdb.org` | InfluxDB organization | `""` |
| `influxdb.bucket` | InfluxDB bucket | `""` |
| `persistence.enabled` | Enable persistent storage | `true` |
| `persistence.size` | Size of persistent volume | `100Mi` |
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `512Mi` |

## Upgrading

```bash
helm upgrade anker-solix-exporter . \
  -f my-values.yaml \
  -n monitoring
```

## Uninstalling

```bash
helm uninstall anker-solix-exporter -n monitoring
```

**Note:** This will not delete the PVC. To delete it manually:

```bash
kubectl delete pvc anker-solix-exporter-data -n monitoring
```

## Troubleshooting

### Check Pod Status

```bash
kubectl get pods -n monitoring
kubectl logs -n monitoring deployment/anker-solix-exporter
```

### Check Resume State

```bash
kubectl exec -n monitoring deployment/anker-solix-exporter -- cat /data/last_export.json
```

### Test InfluxDB Connection

```bash
kubectl exec -n monitoring deployment/anker-solix-exporter -- \
  wget -O- http://influxdb:8086/health
```

## Development

### Linting

```bash
helm lint .
```

### Testing

```bash
helm template anker-solix-exporter . -f values.yaml > test-output.yaml
kubectl apply --dry-run=client -f test-output.yaml
```

### Packaging

```bash
helm package .
```
