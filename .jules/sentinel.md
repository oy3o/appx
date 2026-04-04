
## 2024-05-18 - [Defense in Depth] Missing Security Headers
**Vulnerability:** The Appx HTTP container was not injecting baseline security headers (e.g., `X-Content-Type-Options`, `X-Frame-Options`, `Strict-Transport-Security`), leaving downstream applications vulnerable to MIME-sniffing, clickjacking, and protocol downgrade attacks by default.
**Learning:** Container-level application frameworks should provide secure defaults. Relying on downstream developers to manually add basic security headers often leads to omission. Injecting these headers at the container's HTTP middleware layer ensures ubiquitous baseline protection.
**Prevention:** Always implement a default security middleware in custom HTTP frameworks to inject fundamental protection headers.
