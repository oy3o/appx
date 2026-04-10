## 2026-04-10 - Missing Default Security Headers
**Vulnerability:** The HTTP service lacked default security headers like X-Content-Type-Options, X-Frame-Options, and Strict-Transport-Security.
**Learning:** In a Go-based service container, security headers must be explicitly injected in the middle layer before routing logic, and HTTP/3 support requires special handling to preserve early response modification.
**Prevention:** Implement a default `securityHeadersMiddleware` in the HTTP service pipeline to automatically append these critical response headers without developer intervention.
