## 2024-05-20 - Missing Default Security Headers
**Vulnerability:** The production-ready HTTP container (`appx`) lacked fundamental baseline security headers (e.g., `X-Content-Type-Options`, `X-Frame-Options`, `Strict-Transport-Security`).
**Learning:** Even full-featured application containers might omit HTTP-level security headers, trusting the reverse proxy (which may or may not exist).
**Prevention:** Implement a default middleware that unconditionally injects defense-in-depth security headers like `nosniff`, `DENY` for frames, and HSTS when TLS is enabled.
