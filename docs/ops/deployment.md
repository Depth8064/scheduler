# Deployment

## Primary Deployment Path

Docker is the primary deployment path for all environments (dev, test, prod). Non-container process deployments are secondary and only for local debugging.

## Topology

1. Reverse proxy (TLS termination).
2. Scheduler app container.
3. PostgreSQL database.

## Docker Standards

1. Build a single production image per commit (multi-stage Dockerfile, minimal runtime base).
2. Run database migrations as part of container startup gating before serving traffic.
3. Publish immutable image tags:
	- git SHA tag (required)
	- semver tag (release builds)
	- `latest` only for dev branches/non-production use
4. Define healthcheck endpoint contract (`/healthz`) in image/runtime manifests.
5. Enforce non-root container user and read-only root filesystem where practical.
6. Configure structured logs to stdout; runtime handles retention/aggregation.
7. Keep environment-specific config in deployment manifests, never baked into image.

## Orchestration Plan (V1)

1. Local/dev path: docker compose stack with app + postgres.
2. CI path: ephemeral containerized integration test stack.
3. Production path: container platform deployment (Kubernetes, ECS, or equivalent) with rolling updates and health-gated rollout.

## Runtime Configuration

Required environment variables:

1. SCHEDULER_LISTEN_ADDR
2. SCHEDULER_DATABASE_URL
3. SCHEDULER_SESSION_SECRET
4. JB2_CLIENT_ID
5. JB2_CLIENT_SECRET
6. JB2_BASE_URL (optional override)
7. JB2_TOKEN_URL (optional override)

## Production Baselines

1. Run behind TLS-enabled reverse proxy.
2. Use dedicated DB user with least privileges.
3. Enable daily DB backups and restore test cadence.
4. Configure health checks for app and database connectivity.
5. Keep logging configuration centralized in application bootstrap and version-controlled with deployment manifests.
6. Set production logging with stdout enabled and debug/verbose disabled by default.
7. If file/syslog sinks are enabled, validate rotation and transport before cutover.
8. Pin image by immutable digest in production manifests.
9. Enforce startup/readiness/liveness probes and fail deployment on unhealthy rollout.
10. Keep rollback path documented and tested at least once per release cycle.
