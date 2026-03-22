## 2024-03-22 - [Security Headers Middleware]
**Vulnerability:** Missing baseline security headers in HTTP responses.
**Learning:** The appx HTTP service lacked a middleware to inject standard security headers.
**Prevention:** Implement a `securityHeadersMiddleware` in the default handler chain to inject `X-Content-Type-Options`, `X-Frame-Options`, and `Strict-Transport-Security`.
