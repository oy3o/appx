## 2024-04-18 - [Hardcoded Credentials in Example Code]
**Vulnerability:** Example code and documentation (README) contained hardcoded credentials ("admin"/"s3cret") for the Monitor Service authentication middleware.
**Learning:** Developers often copy-paste example code directly into production. If examples contain hardcoded secrets without a fail-secure mechanism, those secrets become built-in backdoors. We must treat examples and documentation with the same security rigor as production code.
**Prevention:** Never hardcode credentials in examples or documentation. Always use environment variables (`os.Getenv`) or configuration files (`viper`, struct fields) combined with a strict fail-secure check (e.g., rejecting if credentials are empty strings).
