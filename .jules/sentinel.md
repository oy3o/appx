## 2024-05-18 - [Hardcoded Credentials]
**Vulnerability:** Found hardcoded credentials `admin` and `s3cret` in example/main.go and README.md monitor authentication.
**Learning:** Hardcoded credentials should never be in source code, even for examples, as they can be easily copied and deployed to production.
**Prevention:** Use environment variables or configuration files to load credentials.
