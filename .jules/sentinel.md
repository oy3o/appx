## 2024-05-18 - Missing Content-Type Headers on Plain Text Endpoints
**Vulnerability:** The `/healthz` endpoints returned plain text ("OK" or "ok") but didn't set a `Content-Type` header, which could allow browsers to potentially perform MIME sniffing if attackers manipulate the response.
**Learning:** Default fallback HTTP handlers often miss security headers like `Content-Type: text/plain`, leaving endpoints vulnerable to basic web attacks when content isn't explicitly defined.
**Prevention:** Always explicitly define the `Content-Type` header (e.g., `text/plain; charset=utf-8`) for non-HTML/JSON HTTP endpoints, even if the payload is simple text like "ok".
