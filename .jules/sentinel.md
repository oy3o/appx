## 2026-04-09 - [Security Headers]
**Vulnerability:** Missing security headers (X-Content-Type-Options, X-Frame-Options, HSTS) in HTTP responses.
**Learning:** Security headers should be injected as a middleware in the default handler chain before application logic to protect against common web vulnerabilities like MIME-sniffing and Clickjacking.
**Prevention:** Apply a securityHeadersMiddleware in the HTTP server configuration.
