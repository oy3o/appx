## 2024-05-18 - [Add Security Headers]
**Vulnerability:** Missing security headers (X-Content-Type-Options, X-Frame-Options, HSTS).
**Learning:** Adding baseline security headers by default to the framework ensures all services created with Appx inherit defense-in-depth mechanisms for common web vulnerabilities like MIME sniffing, clickjacking, and man-in-the-middle attacks.
**Prevention:** Include a security headers middleware in the default handler chain.
