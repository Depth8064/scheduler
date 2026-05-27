# Security Model

## Authentication

1. V1 uses local accounts with secure password storage.
2. Session-based auth with HttpOnly, Secure cookies.
3. Session expiry is configured via environment variable and enforced server-side. Default maximum session lifetime is 8 hours.
4. Login endpoint enforces rate limiting (e.g. 5 failed attempts per 15 minutes per IP) to prevent brute force.
5. Future OIDC support will be added behind the same auth interface.

## Authorization

1. Roles: admin and workstation.
2. Workstation users are server-side scoped to allowed workstation IDs. Requests for data outside that scope are rejected 403.
3. Admin users have full schedule access.

## CSRF Protection

1. All mutating requests (POST, PATCH, DELETE) from the browser require a CSRF token.
2. The server issues a CSRF token as a non-HttpOnly cookie on session start.
3. The client reads it and sends it as a request header (e.g. X-CSRF-Token).
4. The server validates the token matches the session before processing any state-changing operation.

## Transport and Edge

1. TLS termination at reverse proxy.
2. Proxy forwards only required headers to the app.
3. App listens on private interface/network behind proxy.

## Secret Management

1. No secrets in source control.
2. Secrets loaded from environment variables.
3. Production secrets injected by deployment platform.

## Audit Requirements

1. Log all schedule changes and execution status updates.
2. Log all sync runs and import errors.
3. Keep immutable event history.
