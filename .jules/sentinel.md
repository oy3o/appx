## 2023-10-25 - [Missing Baseline Security Headers]
**Vulnerability:** The appx HTTP service wrapper exposed endpoints without baseline security headers (X-Content-Type-Options, X-Frame-Options, Strict-Transport-Security), leaving it vulnerable to MIME-sniffing, clickjacking, and man-in-the-middle attacks.
**Learning:** Even internal or behind-reverse-proxy HTTP services should enforce baseline security headers as defense-in-depth, especially when TLS is terminated directly by the service (as supported by `cert.Manager`).
**Prevention:** Implement a default `securityHeadersMiddleware` in the HTTP service's handler chain that injects these headers automatically for all routes.
