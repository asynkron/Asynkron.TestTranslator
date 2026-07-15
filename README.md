# testtranslator

A single self-contained Go binary that translates test-result and coverage
artifacts from many runners and coverage tools into two widely supported output
formats:

- **Test results** become **JUnit XML**.
- **Coverage** becomes **Cobertura XML**.

It works out of the box in local development and CI without requiring Python,
Node.js, Java, or .NET to be installed. No runner-specific converter programs
are needed.

## Design principles

testtranslator uses **explicit, deterministic input adapters**. It never guesses
a format from a filename, never uses confidence scores, and never falls back to
heuristic parsing:

- Every input is selected by an exact format identifier.
- A format declared incorrectly fails with a clear error.
- Source information that cannot be represented in the output is reported, never
  silently discarded.
- Unsafe, escaping, or unrootable source paths are rejected.
- Output is deterministic with stable ordering.
- Generated JUnit and Cobertura documents are validated before they replace the
  destination file.
- Output is written atomically, so a failed conversion never leaves a partial
  file.
- Diagnostics go to stderr (or a JSON stream); artifacts go to stdout or the
  requested output file.
- A non-zero exit status is preserved for malformed input, unsupported data, or
  conversion failure.

## Install

```sh
go install github.com/asynkron/testtranslator/cmd/testtranslator@latest
```

## Usage

```sh
# Test result conversion
testtranslator results \
  --format go-test-json \
  --input go-test.jsonl \
  --output junit.xml

# Coverage conversion
testtranslator coverage \
  --format go-coverprofile \
  --input cover.out \
  --repo-root . \
  --go-module github.com/example/project \
  --output cobertura.xml

# Multiple explicitly typed inputs merged into one JUnit document
testtranslator results \
  --input go-test-json=reports/go.jsonl \
  --input vstest-trx=reports/backend.trx \
  --output junit.xml

# Coverage from several sources (overlapping files require --merge union)
testtranslator coverage \
  --input go-coverprofile=reports/cover.out \
  --input istanbul-json=reports/coverage-final.json \
  --repo-root . \
  --output cobertura.xml
```

Inputs and outputs support files and stdin/stdout (use `-`). List every bundled
adapter:

```sh
testtranslator formats            # both pipelines
testtranslator results formats    # test-result adapters
testtranslator coverage formats   # coverage adapters
```

Validate an existing artifact:

```sh
testtranslator validate junit --input junit.xml
testtranslator validate cobertura --input coverage.xml
```

Emit machine-readable diagnostics:

```sh
testtranslator results --format go-test-json --input go.jsonl --output junit.xml \
  --diagnostics json
```

Build an optional bundle manifest that references both artifacts with checksums
(JUnit and Cobertura remain separate standard files — coverage is never embedded
in JUnit):

```sh
testtranslator manifest \
  --add test-results:junit:junit.xml \
  --add coverage:cobertura:coverage.xml \
  --output test-artifacts.json
```

## Supported formats

### Test results → JUnit XML

| Format id | Aliases | Source |
|-----------|---------|--------|
| `go-test-json` | `gotest`, `go-json` | Go `go test -json` / test2json |
| `vstest-trx` | `trx`, `mtp-trx` | Visual Studio Test Platform / Microsoft.Testing.Platform TRX |
| `nunit` | `nunit2`, `nunit3` | NUnit 2 and NUnit 3 XML |
| `xunit` | `xunit2`, `xunit3`, `dotnet-xunit` | xUnit.net v2 and v3 XML |
| `junit-xml` | `surefire`, `gradle-junit`, `sbt-junit`, `pytest-junit`, `phpunit`, `cargo-nextest`, `erlang-ct`, `junit4` | JUnit-family XML |
| `tap` | `tap12`, `tap13` | Test Anything Protocol 12 and 13 |

### Coverage → Cobertura XML

| Format id | Aliases | Source |
|-----------|---------|--------|
| `go-coverprofile` | `gocover`, `gocoverage` | Go coverprofile (set, count, atomic) |
| `istanbul-json` | `istanbul`, `nyc`, `vitest-json` | Istanbul `coverage-final.json` |
| `lcov` | `lcov-info` | LCOV tracefiles |
| `cobertura` | `cobertura-xml`, `coveragepy-cobertura` | Cobertura XML (normalization) |
| `jacoco` | `jacoco-xml` | JaCoCo XML |

Run `testtranslator formats` for the authoritative, always-current list.

## Documentation

- [Architecture](docs/architecture.md)
- [Authoring an adapter](docs/adapters.md)
- [Third-party notices](THIRD_PARTY_NOTICES.md)

## License

Apache License 2.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE).
