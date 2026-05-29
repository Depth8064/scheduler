# Scheduler

A Go scheduler service for JobBOSS integration.

## Local development

Copy `.env.example` to `.env` and update values as needed.

Use the provided Docker Compose example:

```bash
cp .env.example .env
docker compose -f docker-compose.example.yml up --build
```

The compose example runs the application and a local PostgreSQL container. The application reads settings from `.env.example` and overrides the database URL with the compose-managed DB.

## Container health checks

The scheduler binary supports a built-in health check mode that does not require `curl` inside the image:

```bash
/scheduler --healthcheck
```

It performs an HTTP GET against the service's own `/healthz` endpoint using the configured listen address and exits with:

- `0` when healthy
- `1` when unhealthy

Compose example:

```yaml
healthcheck:
	test: ['CMD', '/scheduler', '--healthcheck']
	interval: 10s
	timeout: 5s
	retries: 5
	start_period: 20s
```

## CI/CD

GitHub Actions is configured in `.github/workflows/ci.yml`.

- `test` job runs on all pushes to `main` and pull requests to `main`
- `publish` job runs on pushes to `main` and version tags like `v1.2.3`
- On `main`, the Docker image is published to GitHub Container Registry as `ghcr.io/<owner>/scheduler:main`
- On semver tag pushes, the image is published with full `x.y.z` tag plus floating `x.y` and `x` tags

The workflow also uploads the compiled binary as a workflow artifact.
