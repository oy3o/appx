## 2024-05-24 - [Avoid heap allocation when writing static strings]
**Learning:** Casting string literals to byte slices (e.g. `w.Write([]byte("OK"))`) when writing to an interface like `http.ResponseWriter` forces a heap allocation for every request.
**Action:** Use `io.WriteString(w, "string")` instead, which leverages the `io.StringWriter` interface implementation in the underlying response writer to avoid the escaping heap allocation.
