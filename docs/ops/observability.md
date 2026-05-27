# Observability

## Logging

1. Internal logging is powered by the shared ring-buffer Manager in internal/logging.
2. Structured JSON logs in production.
3. Console output is human-friendly and supports verbose formatting in development.
4. Include request_id, user_id, workstation_id when available.
5. Emit explicit event names for sync and scheduling actions.
6. Prefer stable category values (for example http.request, sync.run, scheduling.update).
7. Use LogRequest for all HTTP access logging.

## Logging Behavior and Operations

1. Logger fan-out supports stdout, file, and remote syslog sinks.
2. The ring uses drop_oldest behavior under pressure and tracks per-sink missed counters.
3. Overflow warnings are throttled and emitted to stderr.
4. File rotation is size-based and keeps bounded rotated files when configured.
5. Diagnostics must be captured during incidents: ring head/tail usage, per-sink lag, per-sink missed counts.
6. Recovery action for sustained missed events: reduce verbose/debug volume, increase sink throughput, or reduce sink count.

## Metrics (Target)

1. HTTP request count/latency/error rate.
2. Sync duration and records processed.
3. Queue depth by workstation.
4. State transition counts.

## Alerts (Target)

1. Repeated sync failures.
2. Elevated API error rates.
3. Database connectivity failures.
