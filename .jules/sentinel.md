## 2024-05-24 - [Fix Hardcoded Credentials in Example]
**Vulnerability:** Found hardcoded string literals `"admin"` and `"s3cret"` being used for HTTP Basic authentication validation within `monitorAuth` in `example/main.go`.
**Learning:** Even inside `example/` directories, hardcoding credentials sets a poor standard, and developers often copy-paste example code directly into production systems, propagating the vulnerability.
**Prevention:** Always use configuration-driven authentication (e.g., environment variables, config files like `viper`/`yaml` maps) in example code, just like you would in production code, to enforce secure-by-default patterns.
