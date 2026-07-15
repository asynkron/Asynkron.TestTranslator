# Authoring an adapter

An adapter turns one exact source format into an internal model. Result adapters
implement `results.Adapter`; coverage adapters implement `coverage.Adapter`.

## Result adapter

```go
package myfmt

import (
	"io"

	"github.com/asynkron/Asynkron.TestTranslator/internal/results"
)

func init() {
	results.Register("my-format", "Human description", Adapter{}, "alias1", "alias2")
}

type Adapter struct{}

func (Adapter) Parse(r io.Reader, opts results.Options) (*results.Report, error) {
	// 1. Bound input: wrap r in an io.LimitedReader.
	// 2. Verify the shape (root element / leading directive). Fail clearly on mismatch.
	// 3. Build suites; append cases with suite.AddCase (it validates invariants).
	// 4. Warn via opts.Diag for any source data JUnit cannot represent.
	// Do NOT sort — the caller sorts the merged report deterministically.
	return report, nil
}
```

## Coverage adapter

```go
func (Adapter) Parse(r io.Reader, opts coverage.Options) (*coverage.Report, error) {
	// 1. Bound input.
	// 2. Verify the shape. Fail clearly on mismatch.
	// 3. Normalize each source path with opts.Paths (repo-relative POSIX, safe).
	// 4. Build coverage.File values; use file.AddLine and file.SetMetric.
	// 5. Preserve native metrics distinctly; mark derived metrics as derived.
	// 6. Call report.Normalize() before returning (sorts + validates).
	return report, nil
}
```

## Rules every adapter must follow

- **Explicit only.** Verify the input shape and error on mismatch. No fuzzy
  detection, no heuristic fallback.
- **No silent loss.** Report unrepresentable source data with `opts.Diag.Warnf`.
- **Bound resources.** Use `io.LimitedReader`; cap scanner tokens. For XML,
  set `dec.Strict = true` and `dec.Entity = map[string]string{}` to disable
  external/custom entities.
- **Safe paths.** Route every source path through `opts.Paths` (coverage). Reject
  escaping or unrootable paths — do not guess.
- **Determinism.** Coverage adapters call `report.Normalize()`. Result adapters
  leave ordering to the caller's `Report.Sort`.
- **Register** the adapter in `init()` with a stable canonical id and add its
  blank import to `internal/adapters/adapters.go`.

## Definition of done

An adapter is supported only when:

- its exact source format and supported versions are documented;
- it is selected through a stable format identifier;
- valid representative fixtures convert successfully;
- malformed and mismatched inputs fail clearly;
- source-specific edge cases have tests;
- conversion losses are reported;
- output is deterministic and validates;
- path handling is safe where paths are present;
- license and upstream provenance are recorded for any reused code or fixtures;
- the public supported-format list (README + `formats` command) includes it.

## Reused code and fixtures

When implementation code is copied or ported from upstream, record the upstream
repository URL, commit/release, copied files, local modifications, license, and
required notices in an `UPSTREAM.md` beside the code, retain copyright headers,
and add an entry to `THIRD_PARTY_NOTICES.md`. Redistributed test fixtures keep
their original license, recorded in the fixture directory's `PROVENANCE.md`.
