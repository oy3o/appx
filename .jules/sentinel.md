## 2024-03-14 - [CRITICAL] Fix ACME HostPolicy DoS Vulnerability
**Vulnerability:** The ACME cert manager was configured with a `nil` `HostPolicy` if no domains were specified, allowing it to attempt TLS certificate issuance for any requested domain in the SNI.
**Learning:** A `nil` `HostPolicy` in `golang.org/x/crypto/acme/autocert` is wildly insecure as it allows all hosts, leading to Let's Encrypt rate limit exhaustion and disk exhaustion (DoS).
**Prevention:** Always use a strict `HostPolicy`, such as `autocert.HostWhitelist`, even if the domain list is empty (which correctly denies all requests).
