## 2026-03-16 - [Optimize Shannon Entropy Calculation]
**Learning:** Shannon entropy calculation (`-sum(p_i * log2(p_i))` where `p_i = count_i / total_length`) inside a loop incurs significant floating point division overhead. This can be mathematically hoisted out of the loop using logarithm properties (`log2(A/B) = log2(A) - log2(B)`). The formula becomes `log2(length) - (sum(count_i * log2(count_i)) / length)`.
**Action:** When calculating Shannon entropy in hot paths (like password strength checking during startup or auth cycles), hoist the division outside the loop mathematically for a ~25% performance gain.

## 2026-03-18 - [Optimize Regex Compilation in Hot Path]
**Learning:** `regexp.MustCompile` is highly expensive and causes significant memory allocations and CPU overhead when called on every check. The Go garbage collector and heap allocation patterns show it explicitly. It should always be executed exactly once per regex.
**Action:** When a regex string is constant and frequently reused, extract it to a package-level global variable to compute it once at module initialization, which leads to ~80% fewer allocations and 6x faster execution time.

## 2026-03-19 - [Replace Regex validation with byte iteration in hot paths]
**Learning:** While `regexp.MustCompile` at initialization avoids re-compilation overhead, the actual `MatchString` execution is still surprisingly expensive. For simple character class validations (like `[a-zA-Z]` or `[0-9]`), a handwritten byte iteration loop avoids internal state machine overhead and is an order of magnitude faster.
**Action:** When performing simple alphanumeric or symbol checks on short strings in a hot path, use direct string index iteration instead of regular expressions.

## 2026-03-20 - [Optimize Array Iteration in Hot Paths]
**Learning:** Iterating over a smaller fixed-size array (like `[128]int` for pure ASCII counting instead of `[256]int`) saves unnecessary loop iterations in hot paths like entropy calculation. Do NOT use byte length checks as an optimization before `strings.EqualFold` because `EqualFold` handles Unicode case-folding where characters might have different byte lengths (e.g., 's' is 1 byte, 'ſ' is 2 bytes but they fold together). Doing so introduces security vulnerabilities by bypassing the check.
**Action:** Size arrays to match exact domain requirements (e.g. 128 for ASCII) rather than generic bounds to save loop iterations. Never optimize `strings.EqualFold` with byte length checks if there's any chance of Unicode input.

## 2026-03-24 - [Optimize Multiple Character Checks]
**Learning:** Large `switch` statements for multiple continuous ASCII ranges (like checking for any symbol or number out of 30+ possibilities) add branching overhead. We can check continuous ranges directly using boolean conditions (e.g., `c >= '!' && c <= '/'`) and `||` operators instead, which map closely to hardware branch predictions and eliminate the switch's jump tables, resulting in ~30% faster execution time in hot loops.
**Action:** Use boolean range checks (`>=` and `<=`) combined with `||` instead of large `switch` statements when checking if an ASCII character falls into multiple continuous blocks in hot paths.

## 2026-03-25 - [Optimize UTF-8 Decoding Overhead in Pure ASCII Strings]
**Learning:** The `for _, c := range s` loop in Go implicitly decodes UTF-8 runes on every iteration, which introduces decoding overhead even if the string is mostly ASCII.
**Action:** In hot paths processing mostly ASCII strings (like passwords), manually iterate over bytes (`s[i]`), check if the byte is an ASCII character (`< utf8.RuneSelf`), and only fallback to `utf8.DecodeRuneInString(s[i:])` when encountering multi-byte characters. This optimization reduces latency by avoiding unnecessary rune decoding for pure ASCII inputs.
