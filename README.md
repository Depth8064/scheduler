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

## CI/CD

GitHub Actions is configured in `.github/workflows/ci.yml`.

- `test` job runs on all pushes to `main` and pull requests to `main`
- `publish` job runs on pushes to `main` and version tags like `v1.2.3`
- On `main`, the Docker image is published to GitHub Container Registry as `ghcr.io/<owner>/scheduler:main`
- On semver tag pushes, the image is published with full `x.y.z` tag plus floating `x.y` and `x` tags

The workflow also uploads the compiled binary as a workflow artifact.
