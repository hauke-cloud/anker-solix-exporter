<!-- llm-readme-management spec=1 commit=dacb44c4843132b07c734599225537ff5eb8bfbe template=terraform model=qwen3.8-27b-q4 digest=f68682b55584 generated=2026-09-08T12:00:17Z -->
<a href="https://hauke.cloud" target="_blank"><img src="https://img.shields.io/badge/home-hauke.cloud-brightgreen" alt="hauke.cloud" style="display: block;" /></a>
<a href="https://github.com/hauke-cloud" target="_blank"><img src="https://img.shields.io/badge/github-hauke.cloud-blue" alt="hauke.cloud Github Organisation" style="display: block;" /></a>
<a href="https://github.com/hauke-cloud/llm-readme-management" target="_blank"><img src="https://img.shields.io/badge/template-terraform-orange" alt="Repository type - terraform" style="display: block;" /></a>


# Template Repository


<img src="https://raw.githubusercontent.com/hauke-cloud/.github/main/resources/img/organisation-logo-small.png" alt="hauke.cloud logo" width="109" height="123" align="right">


<llm header hint="Say whether this is a reusable module or a root module that owns real state.">

This Go daemon polls the unofficial Anker Solix cloud API for solar-site measurements and writes them into a TimescaleDB database. It is a standalone service, not a reusable module, that you deploy as a binary, Docker image, or Helm chart. It is for self-hosting operators who want their home-solar data in their own database.

</llm>


## :book: Description

<llm description>

`anker-solix-exporter` is a Go daemon that polls the unofficial Anker Solix cloud API (EU endpoint) on a configurable interval and writes the resulting solar-site measurements into a PostgreSQL or TimescaleDB database. It is intended for home-solar-system operators who want to bring Anker Solix data into their own stack—Grafana is included in the local `docker-compose.yml`—rather than relying on the Anker web dashboard.

The exporter authenticates with your Anker account credentials, then on each poll cycle fetches the site list, per-site live scene data (PV power, output power, battery power, state of charge), and per-device daily historical energy values from the last resume point. It upserts site and device rows and batch-inserts measurements, running TimescaleDB migrations (hypertable, 7-day compression, 2-year retention) on startup. A JSON resume-state file tracks how far each device has been exported so restarts do not re-import data.

- Polls all sites of an Anker account at a configurable interval (default 15 s).
- Stores live measurements and daily historical energy in TimescaleDB via GORM.
- Tracks per-device resume state to avoid duplicate imports.
- Deployable as a static binary, a Docker image, or a Helm chart.

The repository is part of the `hauke-cloud` organisation and is built from the shared `template-repository` scaffold.

</llm>


## :clipboard: Requirements

<llm requirements hint="Give the Terraform version from .terraform-version and the provider constraints from versions.tf, plus the credentials the providers need.">

- Go 1.25.0 (pinned in `go.mod`; CI and the Dockerfile build with Go 1.26).
- An Anker Solix account (email and password) for the EU region — the API endpoint is hardcoded to `ankerpower-api-eu.anker.com`.
- A TimescaleDB (or PostgreSQL) instance reachable from the exporter, with the TimescaleDB extension installed and DDL privileges for the connecting role.
- Docker, for integration tests (testcontainers), the `docker-compose` stack, and image builds.
- Helm 3 and a Kubernetes cluster (1.19+) when deploying via the Helm chart; a ReadWriteOnce PVC is required when `persistence.enabled` is true.
- `pre-commit` for contributors (mandatory per `CONTRIBUTING.md`).

</llm>


## 🚀 Getting started

<llm getting_started hint="terraform init, plan and apply, with the backend configuration the repository actually uses. Say plainly if apply touches real infrastructure.">

You need an Anker Solix account (email and password) and a running Docker daemon.

1. Clone the repository.

```bash
git clone https://github.com/hauke-cloud/anker-solix-exporter.git
cd anker-solix-exporter
```

2. Download Go module dependencies.

```bash
make deps
```

3. Create a local configuration file, then edit it to set `anker.email`, `anker.password`, and the `database.*` fields to match your TimescaleDB instance.

```bash
cp config.yaml.example config.yaml
```

4. Start the local stack (TimescaleDB, the exporter, and

</llm>


## :airplane: Usage

<llm usage hint="For a reusable module, the central example is a module block with source, version and the required variables filled in from variables.tf. For a root module, show the workflow instead.">

Once you have the source checked out, the three main ways to run the exporter are:

**Local development**

Create a `config.yaml` in your working directory (see `config.yaml.example` for the full annotated list) and start the daemon:

```yaml
anker:
  email: you@example.com
  password: your-anker-password
  country: DE
  poll_interval: 5m

database:
  host: localhost
  port: 5432
  user: exporter
  password: exporter
  database: timescale
  sslmode: disable

exporter:
  log_level: info
```

```bash
make run
```

This invokes `go run ./cmd/exporter -config config.yaml` against the file in the current directory.

**Docker Compose stack**

The repository ships a `docker-compose.yml` that brings up TimescaleDB (`timescale/timescaledb:latest-pg16`), the exporter, and Grafana 10.4.0 together. Start the full stack with:

```bash
docker compose up -d
```

**Helm chart (Kubernetes)**

The chart lives in `deployments/helm/anker-solix-exporter`. Install it into a `monitoring` namespace, supplying your Anker credentials and database connection:

```bash
helm install anker-solix-exporter ./deployments/helm/anker-solix-exporter \
  --create-namespace --namespace monitoring \
  --set anker.email=you@example.com \
  --set anker.password=your-anker-password \
  --set anker.pollInterval=5m \
  --set database.host=timescaledb \
  --set database.user=exporter \
  --set database.password=exporter \
  --set database.name=timescale
```

The chart defaults to `image.repository: ghcr.io/hauke-cloud/anker-solix-exporter` and enables a 100 Mi ReadWriteOnce PVC for the resume-state file.

</llm>


## :wrench: Configuration

<llm configuration hint="A table of the variables in variables.tf: name, type, default, required. Point at variables.tf for the full set and mention outputs.tf if it exists.">

This repository contains no Terraform or OpenTofu code. The `.terraform-version` and `.opentofu-version` files are inherited from the project template; the only related artifact is the inactive `.github/workflows/terraform-deploy.yml.template`.

The exporter is configured through a single CLI flag and a YAML file whose values can each be overridden by an environment variable. The full annotated example lives in `config.yaml.example`.

**CLI flag**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-config` | string | `/etc/anker-solix-exporter/config.yaml` | Path to the YAML configuration file |

**Configuration file / environment variables**

| Key | Type | Default | Required | Env var |
|-----|------|---------|----------|---------|
| `anker.email` | string | — | yes | `ANKER_EMAIL` |
| `anker.password` | string | — | yes | `ANKER_PASSWORD` |
| `anker.poll_interval` | duration | `15s` | no | `ANKER_POLL_INTERVAL` |
| `database.host` | string | `localhost` | yes | `DB_HOST` |
| `database.port` | int | `5432` | no | `DB_PORT` |
| `database.user` | string | — | yes | `DB_USER` |
| `database.password` | string | — | yes | `DB_PASSWORD` |
| `database.database` | string | — | yes | `DB_NAME` |
| `database.sslmode` | string | `disable` | no | `DB_SSLMODE` |
| `exporter.log_level` | string | `info` | no | `LOG_LEVEL` |

Additional keys (`anker.country`, `anker.debug`, `anker.endpoint_limit`, `anker.request_delay`, `database.migrations_path`, `database.sslcert` / `sslkey` / `sslrootcert`, `exporter.resume_file`) are documented in `config.yaml.example`.

**Helm chart**

The chart at `deployments/helm/anker-solix-exporter/values.yaml` mirrors the same settings under `anker.*`, `database.*`, and `exporter.*` keys and adds `persistence.*`, `image.*`, `resources`, and standard Kubernetes scheduling fields. See `values.yaml` for the complete list.

</llm>


## :hammer: Development

<llm development hint="Cover terraform fmt, validate, tflint and terraform-docs where the repository configures them.">

Before pushing, install and run the pre-commit hooks:

```bash
pre-commit install
pre-commit run --all-files
```

CI also validates that pull-request titles follow the Conventional Commits format (`.github/workflows/pr-title.yml`); a non-conforming title will fail the PR.

**Tests**

Unit tests (no external services needed):

```bash
make test-unit
```

Integration tests spin up a TimescaleDB container via testcontainers, so Docker must be running:

```bash
make test-integration
```

CI runs both: `go test -v -race -short` for unit and `go test -v -race` for integration.

**Lint and format**

```bash
make lint
```

This runs `go vet ./...` and `gofmt -s -w .`. CI additionally runs `staticcheck` (latest) via `dominikh/staticcheck-action@v1`.

**Helm chart**

If you touch the chart, lint and package it before committing:

```bash
make helm-lint
make helm-package
```

CI packages the chart and pushes it to `oci://ghcr.io/hauke-cloud/charts`.

**Dependencies**

After adding or removing Go modules:

```bash
make deps
```

This runs `go mod download` and `go mod tidy`; commit the updated `go.mod` and `go.sum`.

</llm>


## 📄 License

This Project is licensed under the GNU General Public License v3.0

- see the [LICENSE](LICENSE) file for details.


## :coffee: Contributing

To become a contributor, please check out the [CONTRIBUTING](CONTRIBUTING.md) file.


## :email: Contact

For any inquiries or support requests, please open an issue in this
repository or contact us at [contact@hauke.cloud](mailto:contact@hauke.cloud).
