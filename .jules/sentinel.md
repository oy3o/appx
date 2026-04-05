## 2024-05-14 - [Sentinel] Missing Security Headers
**Vulnerability:** Missing default security headers in HTTP service (X-Content-Type-Options, X-Frame-Options, Strict-Transport-Security)
**Learning:** HTTP services in this framework default to not injecting baseline security headers.
**Prevention:** Add a `securityHeadersMiddleware` by default for the `appx` HTTP service to inject standard security headers.
