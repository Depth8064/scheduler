# Backend Structure

## Package Layout

1. cmd/scheduler: application entrypoint and wiring.
2. internal/config: environment configuration and validation.
3. internal/logging: structured logger setup.
4. internal/httpapi: HTTP router, middleware, handlers.
5. internal/db: migration runner and DB bootstrap.
6. internal/store: persistence adapters and repositories.
7. internal/jobbosssync: ingestion services (to be implemented).
8. internal/scheduling: scheduling domain services (to be implemented).
9. internal/execution: progress/event services (to be implemented).
10. internal/auth: account/session services (to be implemented).

## Dependency Rules

1. cmd can import any internal package.
2. httpapi can depend on service-layer packages.
3. domain/service packages must not depend on httpapi.
4. store packages must not depend on httpapi.
5. integration adapters are called via service layer, not directly in handlers.

## Coding Standards

1. Context from incoming request must be propagated.
2. Errors are wrapped with operation context.
3. Logging uses structured key-value pairs.
4. Public API responses use stable JSON contracts.

## Logging Patterns (Canonical)

1. Use internal/logging Manager directly via logging.GetManager().
2. Do not introduce compatibility shims or alternate logger wrappers.
3. Configure logging once at startup with logging.DefaultConfig() followed by explicit overrides per environment.
4. Use manager methods for application logs: Info, Warn, Error, Debug, Verbose.
5. Use LogRequest for HTTP request logs so duration and status are normalized.
6. Use category and stable field keys for machine-readable events.
7. Keep Debug and Verbose disabled in production unless incident response requires temporary elevation.
8. Use VerboseField for console-only detail that should not clutter non-verbose output.
9. Keep sink configuration centralized in bootstrap, not in handlers/services.
10. Prefer manager diagnostics for logger health checks when adding admin diagnostics endpoints.
