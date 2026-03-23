## 2026-03-23 - [HTTP Security Headers Injection]
**Vulnerability:** Missing baseline security headers in HTTP responses by default. This makes the application potentially susceptible to basic MIME-type sniffing, clickjacking, and MITM attacks via TLS stripping.
**Learning:** Default HTTP handler implementations often neglect security headers. These should be natively bundled via a default middleware in the service's handler chain before hitting the final business handler, especially when dealing with TLS and HTTP/3 support.
**Prevention:** Always implement a default middleware injecting `X-Content-Type-Options`, `X-Frame-Options`, and `Strict-Transport-Security` headers on initialization. Provide the option to allow users to override them if needed, but the default state must be secure.
