## 2024-05-24 - Missing Security Headers in Default HTTP Service
**Vulnerability:** The default `HttpService` in `appx` lacked baseline security headers like `X-Content-Type-Options`, `X-Frame-Options`, and `Strict-Transport-Security`, potentially exposing endpoints to MIME-sniffing and clickjacking attacks.
**Learning:** Security by default means every HTTP server exposed by a library should carry fundamental security headers in its default middleware chain, not leave it entirely up to the user to configure.
**Prevention:** In Go HTTP server libraries, always implement a built-in `securityHeadersMiddleware` that gets injected automatically to enforce safe defaults (e.g. `nosniff`, `DENY` frames, and `HSTS` when TLS is enabled).
