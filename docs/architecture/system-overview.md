# System Overview

## Architecture Summary

Scheduler is a Go web application with a custom HTML/JS frontend and PostgreSQL persistence.

1. JobBOSS is integrated in read-only mode via go-jobboss2-api.
2. A reverse proxy terminates TLS and forwards traffic to Scheduler.
3. Scheduler stores scheduling, progress, and audit data as the local system of execution.

## High-Level Flow

1. Admin triggers sync or scheduled sync runs.
2. Scheduler pulls pending data from JobBOSS API.
3. Scheduler upserts mirrored external records and updates local scheduling entities.
4. Admin assigns and orders work per workstation.
5. Workstation users execute and update progress.
6. All mutations are persisted to audit/event history.

## Trust Boundaries

1. External boundary: reverse proxy and TLS edge.
2. App boundary: authenticated API and session validation.
3. Data boundary: PostgreSQL with least-privilege credentials.
4. Integration boundary: outbound read-only JobBOSS access.
