## 2026-03-16 - [Optimize Shannon Entropy Calculation]
**Learning:** Shannon entropy calculation (`-sum(p_i * log2(p_i))` where `p_i = count_i / total_length`) inside a loop incurs significant floating point division overhead. This can be mathematically hoisted out of the loop using logarithm properties (`log2(A/B) = log2(A) - log2(B)`). The formula becomes `log2(length) - (sum(count_i * log2(count_i)) / length)`.
**Action:** When calculating Shannon entropy in hot paths (like password strength checking during startup or auth cycles), hoist the division outside the loop mathematically for a ~25% performance gain.

## 2026-03-18 - [Optimize Regex Compilation in Hot Path]
**Learning:** `regexp.MustCompile` is highly expensive and causes significant memory allocations and CPU overhead when called on every check. The Go garbage collector and heap allocation patterns show it explicitly. It should always be executed exactly once per regex.
**Action:** When a regex string is constant and frequently reused, extract it to a package-level global variable to compute it once at module initialization, which leads to ~80% fewer allocations and 6x faster execution time.
