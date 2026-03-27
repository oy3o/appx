## 2024-05-24 - [Baseline Security Headers]
**Vulnerability:** Missing default security headers (X-Content-Type-Options, X-Frame-Options, Strict-Transport-Security).
**Learning:** The appx HTTP service didn't enforce baseline security headers, leaving applications potentially vulnerable to MIME sniffing, clickjacking, and man-in-the-middle attacks over insecure connections.
**Prevention:** Implement a `securityHeadersMiddleware` in the default HTTP handler chain to ensure these critical headers are always injected by default, especially HSTS when TLS is enabled.