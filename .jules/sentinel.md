## 2026-03-16 - [Information Disclosure] Health Check Error Leak
**Vulnerability:** Health handler exposed internal errors (like DB connection failures) to clients via `fmt.Sprintf("Health check failed: %v", err)`.
**Learning:** Returning detailed errors on public monitoring endpoints can accidentally leak internal network topology, IPs, or connection string details to unauthenticated users.
**Prevention:** Always fail securely by logging detailed errors internally but returning generic error messages (e.g., "Health check failed") to external clients.
