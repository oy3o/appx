## 2026-05-02 - [CRITICAL] Hardcoded credentials and timing attack vulnerability
**Vulnerability:** Hardcoded credentials ('admin':'s3cret') in authentication middleware examples and use of standard string equality (==) for secret comparison.
**Learning:** Hardcoded credentials in examples are frequently copy-pasted into production, creating severe vulnerabilities. Standard string comparisons leak timing information, allowing attackers to guess passwords byte-by-byte.
**Prevention:** Always read credentials from environment variables or configuration files. Use `crypto/subtle.ConstantTimeCompare` for all sensitive string comparisons. Include fail-secure checks to reject unconfigured environments.
