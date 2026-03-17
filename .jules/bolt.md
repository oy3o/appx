## 2026-03-16 - [Optimize Shannon Entropy Calculation]
**Learning:** Shannon entropy calculation (`-sum(p_i * log2(p_i))` where `p_i = count_i / total_length`) inside a loop incurs significant floating point division overhead. This can be mathematically hoisted out of the loop using logarithm properties (`log2(A/B) = log2(A) - log2(B)`). The formula becomes `log2(length) - (sum(count_i * log2(count_i)) / length)`.
**Action:** When calculating Shannon entropy in hot paths (like password strength checking during startup or auth cycles), hoist the division outside the loop mathematically for a ~25% performance gain.

## 2026-03-17 - [Optimize Calculate Entropy ASCII Fast Path]
**Learning:** `for _, c := range s` iterates over a string by decoding it into UTF-8 runes. This introduces overhead. If a string is purely ASCII (common for passwords and secrets), iterating over it as a byte slice (`for i := 0; i < len(s); i++`) is faster and completely avoids rune decoding.
**Action:** When processing strings that are primarily ASCII in hot paths, add a quick ASCII check and a byte-iteration fast path to bypass UTF-8 decoding overhead.
