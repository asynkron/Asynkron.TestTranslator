# Architecture

testtranslator is organized around two independent pipelines that share the same
principles: explicit format selection, deterministic output, safe path handling,
validation before write, and atomic replacement.

```text
test-result file -> exact source adapter -> internal result model -> JUnit XML
coverage report  -> exact source adapter -> internal coverage model -> Cobertura XML
```

The internal models are implementation details. JUnit XML and Cobertura XML are
the public output contracts.

## Package layout

```text
cmd/testtranslator/            minimal main: wires stdio into the CLI
internal/cli/                  CLI parsing, command dispatch, IO — no conversion logic
internal/adapters/             blank-imports every adapter so init() registers them
internal/pathutil/             repo-relative POSIX normalization and path-safety
internal/diagnostics/          structured warnings/notes (text and JSON rendering)
internal/atomicio/             validate-then-atomic-rename output writer
internal/manifest/             optional bundle manifest with checksums

internal/results/              internal result model + adapter registry
internal/results/junit/        JUnit writer and independent validator
internal/results/gotest/       go test -json / test2json
internal/results/trx/          Visual Studio / Microsoft.Testing.Platform TRX
internal/results/nunit/        NUnit 2 and 3
internal/results/xunit/        xUnit.net v2 and v3
internal/results/surefire/     JUnit-family XML (Surefire, Gradle, pytest, ...)
internal/results/tap/          TAP 12 and 13

internal/coverage/             internal coverage model + adapter registry + merge
internal/coverage/cobertura/   Cobertura writer and independent validator
internal/coverage/coberturain/ Cobertura XML input/normalization
internal/coverage/gocover/     Go coverprofile
internal/coverage/istanbul/    Istanbul coverage-final.json
internal/coverage/lcov/        LCOV
internal/coverage/jacoco/      JaCoCo XML

internal/integration/          proving harness over testdata/real
testdata/real/                 real third-party fixtures (see PROVENANCE.md files)
```

## Key decisions

### Explicit adapters, no auto-detection

Every input is selected by an exact format identifier (`--format` or a
`format=path` input declaration). There is no filename sniffing, no confidence
scoring, and no heuristic fallback. A mismatched format fails with a clear error
because each adapter verifies the shape of its input (root element, leading
directive, event schema) before converting.

### Models that cannot hold contradictory state

`results.TestSuite.AddCase` and `coverage.File.AddLine` validate on construction:
a failed test case must carry a failure; a passed one must not; line numbers are
positive; covered counts never exceed totals. Invalid states cannot be
represented, so writers never have to defend against them.

### Determinism

`Report.Sort` (results) and `Report.Normalize` (coverage) impose stable ordering
of suites, cases, files, and lines. Durations and rates use fixed formatting. No
wall-clock timestamp is embedded in generated Cobertura. Repeated runs over the
same input produce byte-identical output.

### Path safety

`pathutil.Normalizer` converts any source path to a repository-relative POSIX
path. Absolute paths require `--repo-root`; import-path coverprofiles require
`--go-module`. Paths that escape the repository root, or that cannot be rerooted
unambiguously, are rejected rather than guessed.

### Validate, then write atomically

Every generated document is parsed by an **independent** validator (separate
serialization structs from the writer) that checks structural and semantic
invariants — including that declared counts match actual counts. Only after
validation succeeds does `atomicio` write to a temp file in the destination
directory and rename it into place, so a failed conversion never leaves a partial
or corrupt file. Existing output is preserved on failure.

### Diagnostics vs artifacts

Diagnostics (lossy-mapping warnings, derived-metric notes) go to stderr, or to a
machine-readable JSON stream with `--diagnostics json`. Generated artifacts go to
stdout or the requested output file. The two streams never mix.

### Resource bounds

Adapters read through an `io.LimitedReader`, cap scanner token sizes, and disable
unsafe XML features (external/custom entity resolution) on every XML decoder.

## Metric fidelity

The coverage model preserves statements, lines, branches, functions, methods,
instructions, and regions as **distinct** metrics. Derived metrics (for example,
per-line coverage computed from Go statement blocks) are flagged as derived and
carry a documented derivation rule, so a consumer can tell native data from
computed data. Cobertura's line/branch model is produced from these without
mislabeling one metric as another.
