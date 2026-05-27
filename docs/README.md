# Scheduler Documentation Canon

This folder is the source of truth for product and implementation decisions.

## Authority Rules

1. Docs in this folder are canonical for architecture, data model, API behavior, and operational policy.
2. Code changes must reference the doc section they implement.
3. If code and docs conflict, update docs first (or in the same change) before merging code.
4. V1 scope excludes writing data back to JobBOSS.
5. Logging standard is the shared internal/logging Manager and its documented patterns.

## Document Index

- [Product Vision](product/vision.md)
- [Current State and Legacy Spreadsheet](product/current-state.md)
- [Legacy Schedule Reference](product/legacy-schedule-reference.md)
- [System Overview](architecture/system-overview.md)
- [Backend Structure](architecture/backend-structure.md)
- [Data Model](architecture/data-model.md)
- [JobBOSS Integration](architecture/integration-jobboss.md)
- [Security Model](architecture/security.md)
- [API Contract Baseline](api/openapi.yaml)
- [Workstation Flows](api/workstation-flows.md)
- [Admin Flows](api/admin-flows.md)
- [Deployment](ops/deployment.md)
- [Observability](ops/observability.md)
- [Runbooks](ops/runbooks.md)
- [Testing Strategy](testing/strategy.md)
- [V1 Roadmap](roadmap/v1-plan.md)

Notes:
- Scaffolding added in code for `schedule_items` store, execution store, worker skeleton, and CI. Expand these components to implement full behavior per the API docs.
