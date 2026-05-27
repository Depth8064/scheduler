# Testing Strategy

## Quality Bar

1. No direct merges to main without green CI.
2. Every PR includes or updates tests for behavior changes.
3. Bug fixes require a regression test that fails before fix and passes after.
4. API contract changes must update both tests and OpenAPI docs in the same PR.

## Test Layers

1. Unit tests for domain logic and validation.
2. Integration tests for repositories and migrations.
3. API tests for auth, RBAC, and key workflows.
4. Containerized end-to-end smoke tests for startup + health + core auth flow.

## Minimum Coverage Expectations

1. New business logic packages should target meaningful branch coverage (guideline: 80%+ where practical).
2. Critical paths (auth, RBAC, scheduling transitions, sync idempotency) require explicit happy-path and failure-path tests.
3. Migration suites must validate both up and down compatibility where rollback scripts exist.

## Critical Acceptance Tests

1. Workstation user cannot access other workstation queues.
2. Repeated sync runs are idempotent.
3. Concurrency conflict returns HTTP 409 on stale version.
4. Complete flow from import to schedule completion is auditable.
5. HTTP request middleware logs status, path, method, and duration through Manager.LogRequest.
6. Logger remains non-blocking under burst load and application request handling continues.
7. Logger diagnostics counters update correctly when sink lag or overflow is induced in test harness.

## Logging-Specific Test Coverage

1. Unit tests for logging configuration defaults and environment overrides.
2. Unit tests for level filtering behavior: DebugEnabled and VerboseEnabled.
3. Integration tests for startup Configure and shutdown Close lifecycle.
4. Rotation tests validating max file behavior when file sink is enabled.
5. Resilience tests ensuring syslog sink failure does not crash request handling.

## CI Baseline

1. Formatting and static checks:
	- `go fmt ./...` (enforced in CI)
	- `go vet ./...`
2. Unit and integration tests:
	- `go test ./...`
3. Migration validation:
	- apply migrations on clean database
	- start app and verify `/healthz`
4. Docker build validation:
	- build production image from Dockerfile
	- run containerized smoke test against ephemeral postgres
5. Security/dependency hygiene:
	- run govulncheck (or equivalent)
	- fail on high/critical findings unless explicitly waived

## CI Pipeline Stages (Planned)

1. `lint-and-vet`
2. `unit-test`
3. `integration-test-db`
4. `docker-build-and-smoke`
5. `artifact-publish` (main/release branches only)

## Merge and Release Gates

1. Required checks: all stages above green.
2. Required review for changes touching auth, migration files, or scheduling state machine.
3. Release candidates require successful container image build + smoke test on release commit.
