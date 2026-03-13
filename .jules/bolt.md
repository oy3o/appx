## 2024-03-13 - Avoid `sync.RWMutex` locks on hot paths (per-request middlewares)
**Learning:** Calling `http3.Server.SetQUICHeaders` dynamically on every HTTP request created a high-contention lock on `s.mutex.RLock()` across all concurrent requests.
**Action:** When a static or semi-static value (like the `Alt-Svc` header port) is needed on a hot path, precalculate it at startup and inject it via closures to completely avoid the mutex.
