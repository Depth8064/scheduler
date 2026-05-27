# Runbooks

## Sync Failure

1. Check latest sync_runs record and error summary.
2. Validate JobBOSS credentials and network reachability.
3. Retry manual sync.
4. Escalate if two consecutive retries fail.

## Auth Incident

1. Confirm session secret configuration.
2. Validate account lock/disabled state.
3. Rotate session secret if compromise suspected.

## Restore Procedure

1. Restore latest verified backup to staging.
2. Validate schema and row counts.
3. Promote restore process to production only after verification.
