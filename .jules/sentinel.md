## 2024-05-24 - [Init]
**Vulnerability:** Initializing Sentinel journal
**Learning:** Establishing the file for recording critical security learnings.
**Prevention:** N/A
## 2024-05-24 - [Information Leakage in Health Check]
**Vulnerability:** The health check endpoint `/healthz` exposed the specific error message (e.g., "connection refused", "context deadline exceeded") when a check failed.
**Learning:** Exposing detailed error messages in public endpoints can leak sensitive internal architecture details, such as database connection issues or specific service names.
**Prevention:** Always log detailed error messages internally, but return generic error messages to the client. Ensure proper error handling and filtering.
