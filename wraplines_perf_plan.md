# WrapLines performance evaluation plan

## 1) Where `WrapLines` lives

- Implementation: `pkg/tui/components/toolcommon/truncate.go`
  - `func WrapLines(text string, width int) []string`
  - Helper used: `takeRunesThatFit(...)` (same file)
  - Uses `lipgloss.Width(...)` for width checks and a rune-based splitting loop when wrapping is needed.

- Benchmarks already exist:
  - `BenchmarkWrapLines` in `pkg/tui/components/toolcommon/common_test.go`

## 2) What changed recently (in code)

From the current implementation, `WrapLines` includes two performance-oriented changes relative to a naive approach:

- **Fast path**: if `lipgloss.Width(inputLine) <= width`, it appends the line without converting to `[]rune`.
- **Avoid repeated conversions / scanning**: for wrapped lines, converts `inputLine` to `[]rune` once and then advances `start` using `takeRunesThatFit`, which scans forward and ensures progress.

To confirm *what the last commit changed* (and whether it actually touched `WrapLines`), run:

```bash
git log -n 20 -- pkg/tui/components/toolcommon/truncate.go pkg/tui/components/toolcommon/common_test.go

git show --stat HEAD

git show HEAD -- pkg/tui/components/toolcommon/truncate.go
```

If you need a focused diff against the previous commit:

```bash
git diff HEAD~1..HEAD -- pkg/tui/components/toolcommon/truncate.go pkg/tui/components/toolcommon/common_test.go
```

This will tell you whether the last commit modified `WrapLines` itself, its helpers (`takeRunesThatFit`, `runeWidth`), or only added/changed benchmarks.

## 3) Benchmarking approach (before vs after)

### A. Run the existing Go microbenchmarks

The repo already has `BenchmarkWrapLines`. Use it to compare commits.

Run on current commit:

```bash
go test ./... -run '^$' -bench 'BenchmarkWrapLines' -benchmem -count 10
```

Recommended: pin CPU and reduce noise (optional, if available on your system):

- Linux:
  ```bash
  taskset -c 0 go test ./... -run '^$' -bench 'BenchmarkWrapLines' -benchmem -count 10
  ```
- macOS: close background apps; optionally use `GOMAXPROCS=1`:
  ```bash
  GOMAXPROCS=1 go test ./... -run '^$' -bench 'BenchmarkWrapLines' -benchmem -count 10
  ```

### B. Compare `HEAD` vs `HEAD~1` directly

Option 1: manual checkout + run + record output

```bash
# new
git checkout HEAD

go test ./... -run '^$' -bench 'BenchmarkWrapLines' -benchmem -count 10 | tee /tmp/bench_new.txt

# old
git checkout HEAD~1

go test ./... -run '^$' -bench 'BenchmarkWrapLines' -benchmem -count 10 | tee /tmp/bench_old.txt
```

Option 2: use `benchstat` for an apples-to-apples comparison (recommended)

```bash
go install golang.org/x/perf/cmd/benchstat@latest

benchstat /tmp/bench_old.txt /tmp/bench_new.txt
```

### C. Ensure benchmarks reflect real workloads

The existing benchmark subcases are:
- `short_no_wrap`, `short_wrap`, `medium`, `long`, `multiline`

If the reported perf gain/regression is unclear, expand benchmarks (future enhancement):
- Add a case with **wide unicode** (CJK/emoji) where `lipgloss.Width` != rune count.
- Add a case with **many lines** (e.g., 500–5000 lines) to measure allocation behavior.

(If needed, we can add these benchmark cases later, but first run existing ones across commits.)

## 4) What to look for in results

- `ns/op`: primary latency metric.
- `allocs/op` and `B/op`: if changes reduced allocations (e.g., avoiding `[]rune` conversion for the fast path), you should see fewer allocs especially in `short_no_wrap` / lines that fit.
- Regressions might show up in cases dominated by `lipgloss.Width` if it’s expensive; watch `short_no_wrap` and `multiline`.

## 5) Deliverable

- Produce:
  - `bench_old.txt` (previous commit)
  - `bench_new.txt` (current commit)
  - `benchstat` output showing the delta

Then decide: “WrapLines is faster/slower by X% in scenario Y”.
