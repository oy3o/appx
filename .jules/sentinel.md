# Sentinel Security Journal

This journal logs critical security learnings and prevents regressions.

## 2025-04-13 - [Hardcoded Secrets] Removed Hardcoded Credentials
**Vulnerability:** Hardcoded admin:s3cret credentials in example/main.go and README files.
**Learning:** Hardcoded credentials even in examples can be deployed accidentally.
**Prevention:** Always use configuration-driven authentication with fail-secure checks for unconfigured environments.
