## 2025-02-18 - Missing baseline security headers in framework
**Vulnerability:** The Appx HTTP service was not injecting any baseline security headers (like X-Content-Type-Options or X-Frame-Options) despite memory stating it did.
**Learning:** Frameworks often expose bare-bones HTTP handlers. Since Appx intends to be a production-ready container, relying on users to manually set security headers creates a widespread vulnerability across all apps built with the framework.
**Prevention:** Built-in web servers or containers should enforce safe defaults by injecting baseline security headers (`X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Strict-Transport-Security` for TLS) directly into the default middleware chain.
