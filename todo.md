# Asynkron.TestTranslator TODO

## Vision

Build one self-contained Go binary that translates test artifacts from as many
test runners and coverage tools as practical into two widely supported output
formats:

- Test results become JUnit XML.
- Test coverage becomes Cobertura XML.

The tool should work out of the box in local development and CI without
requiring Python, Node.js, Java, .NET, or runner-specific converter programs to
be installed. Existing open-source parsers and converters may be imported,
vendored, copied, or ported when their licenses permit it.

## Non-negotiable behavior

- [ ] Use explicit, deterministic input adapters.
- [ ] Do not use fuzzy detection, confidence scores, or heuristic fallback
      parsing.
- [ ] If a format is declared incorrectly, fail with a clear error.
- [ ] Do not silently discard source information that cannot be represented in
      the output format.
- [ ] Report unsupported and lossy mappings explicitly.
- [ ] Reject unsafe, escaping, or unrootable source paths.
- [ ] Produce deterministic output with stable ordering.
- [ ] Validate generated JUnit and Cobertura documents before replacing the
      destination file.
- [ ] Write output atomically so failed conversion cannot leave a partial file.
- [ ] Keep diagnostic output on stderr and generated artifacts on stdout or in
      the requested output file.
- [ ] Preserve a non-zero exit status for malformed input, unsupported data, or
      conversion failure.

## Command-line experience

The exact command names may evolve, but the primary workflows should remain
simple and scriptable.

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

# Multiple explicitly typed inputs
testtranslator results \
  --input go-test-json=reports/go.jsonl \
  --input vstest-trx=reports/backend.trx \
  --output junit.xml

testtranslator coverage \
  --input go-coverprofile=reports/cover.out \
  --input istanbul-json=reports/coverage-final.json \
  --repo-root . \
  --output cobertura.xml
```

- [ ] Support files and stdin as inputs.
- [ ] Support stdout and files as outputs.
- [ ] Support multiple explicitly typed inputs in one invocation.
- [ ] Add `formats` commands that list every bundled adapter and its aliases.
- [ ] Add `validate` commands for JUnit and Cobertura artifacts.
- [ ] Add a machine-readable diagnostics mode.
- [ ] Consider a manifest-driven mode for repositories with several test
      ecosystems.
- [ ] Never require filename-based auto-detection when `--format` or an
      `format=path` input declaration can be used.

## Test-result pipeline

```text
runner result file -> exact source adapter -> internal result model -> JUnit XML
```

The internal model is an implementation detail. JUnit XML is the public output
contract.

### Initial test-result adapters

- [ ] Go `go test -json` / `test2json`.
  - Reuse `gotest.tools/gotestsum/testjson` where practical.
  - Reuse or adapt gotestsum's JUnit behavior for subtests, interleaved package
    events, build failures, skips, output, panics, and cached results.
  - Preserve the upstream Apache-2.0 license and NOTICE requirements for copied
    code.
- [ ] Visual Studio Test Platform TRX.
- [ ] Microsoft.Testing.Platform TRX.
- [ ] NUnit 2 XML.
- [ ] NUnit 3 XML.
- [ ] xUnit.net v2 XML.
- [ ] xUnit.net v3 XML.
- [ ] JUnit 4 XML.
- [ ] xUnit/JUnit family variants used by pytest.
- [ ] Maven Surefire XML.
- [ ] Gradle JUnit XML.
- [ ] sbt JUnit XML.
- [ ] Erlang Common Test Surefire XML.
- [ ] TAP 12.
- [ ] TAP 13.
- [ ] Jest JSON.
- [ ] Vitest JSON.
- [ ] RSpec JSON.
- [ ] PHPUnit JUnit XML.
- [ ] GoogleTest XML.
- [ ] CTest XML.
- [ ] cargo-nextest JUnit XML.
- [ ] Bazel Build Event Protocol JSON test results.
- [ ] Dart/Flutter machine-readable test output.

### Result conversion rules to define

- [ ] Define the exact suite and testcase naming rule for every adapter.
- [ ] Preserve complete Go subtest paths.
- [ ] Represent package or assembly build failures as deterministic synthetic
      testcases when JUnit has no direct equivalent.
- [ ] Preserve captured stdout and stderr in the appropriate JUnit elements.
- [ ] Define treatment of retries and flaky-test attempts.
- [ ] Define treatment of tests that are discovered but never executed.
- [ ] Define treatment of runner crashes and timeouts.
- [ ] Define whether benchmarks are excluded or included through an explicit
      policy.
- [ ] Define active fuzzing behavior separately from ordinary seed-corpus test
      execution.
- [ ] Preserve source filename and line information when the source format
      provides it.
- [ ] Provide explicit JUnit output profiles only when consumer differences
      require them; do not guess a CI provider.

## Coverage pipeline

```text
coverage report -> exact source adapter -> internal coverage model -> Cobertura XML
```

Cobertura XML is the public coverage output contract. The internal model should
preserve richer native data so conversion losses can be detected and reported.

### Seed implementation from Faktorial

The Faktorial `agentic` repository contains a coverage normalization proof of
concept under:

```text
verticals/experiments/coverage
```

It currently supports:

- Go coverprofiles using `golang.org/x/tools/cover`.
- Vitest/Istanbul `coverage-final.json`.
- Repository-relative POSIX path normalization.
- Rejection and reporting of unsafe or out-of-repository paths.
- Deterministic file ordering.
- Line/statement totals plus optional branch and function totals.

- [ ] Port the useful parser and path-normalization behavior into this
      repository.
- [ ] Preserve tests and representative fixtures.
- [ ] Correctly distinguish the coverage producer from its interchange format;
      for example, Vitest's V8 provider can emit Istanbul-shaped JSON.
- [ ] Improve the model so Go statement counts are not mislabeled as literal
      line counts.
- [ ] Preserve statements, lines, branches, functions, methods, instructions,
      and regions as distinct metrics when supplied by the source.
- [ ] Mark derived metrics as derived and document the exact derivation rule.

### Initial coverage adapters

- [ ] Go coverprofile (`set`, `count`, and `atomic`).
- [ ] Istanbul `coverage-final.json`.
- [ ] LCOV.
- [ ] Cobertura XML input and normalization.
- [ ] JaCoCo XML.
- [ ] OpenCover XML.
- [ ] coverage.py JSON.
- [ ] coverage.py Cobertura-style XML.
- [ ] Coverlet JSON.
- [ ] LLVM coverage JSON.
- [ ] Clover XML.
- [ ] SimpleCov resultset JSON.
- [ ] gcov JSON/intermediate formats.
- [ ] Istanbul/nyc LCOV variants.
- [ ] Rust `cargo llvm-cov` outputs.

### Coverage conversion rules to define

- [ ] Use repository-relative POSIX paths as the normalized file identity.
- [ ] Require `--repo-root` when absolute source paths must be rerooted.
- [ ] Require the Go module path when an import-path coverprofile cannot be
      rooted unambiguously.
- [ ] Reject paths that escape the repository root.
- [ ] Handle Windows drive letters and separators deterministically.
- [ ] Preserve source roots needed by Cobertura consumers.
- [ ] Never synthesize branch or function coverage when absent.
- [ ] Define exact source-map behavior for generated JavaScript and TypeScript.
- [ ] Define treatment of generated code and test files through explicit
      include/exclude configuration.
- [ ] Define merge semantics for repeated files, lines, branches, and test runs.
- [ ] Require explicit merge mode rather than silently combining reports.
- [ ] Record conversion warnings for metrics Cobertura cannot represent.
- [ ] Validate the final Cobertura document against the supported schema and
      semantic invariants.

## Combined test artifacts

JUnit and Cobertura remain separate standard artifacts. A small optional bundle
manifest may reference both without inventing a replacement for either format.

```json
{
  "schema_version": "test-artifacts.v1",
  "artifacts": [
    {
      "kind": "test-results",
      "format": "junit",
      "path": "test-results.junit.xml"
    },
    {
      "kind": "coverage",
      "format": "cobertura",
      "path": "coverage.cobertura.xml"
    }
  ]
}
```

- [ ] Keep bundling optional.
- [ ] Do not embed coverage inside JUnit XML.
- [ ] Allow one manifest to reference multiple result and coverage artifacts.
- [ ] Record source formats, conversion tool version, warnings, and checksums in
      the manifest.

## Architecture

- [ ] Keep `main` minimal.
- [ ] Separate CLI parsing from conversion logic.
- [ ] Define small input-adapter interfaces at their consumption point.
- [ ] Keep source-specific XML and JSON structures inside their adapters.
- [ ] Use streaming parsers where reports can be large.
- [ ] Bound scanner tokens, XML depth, input size, testcase count, and captured
      output size.
- [ ] Disable unsafe XML features and reject malformed XML.
- [ ] Avoid temporary files unless required; clean them reliably when used.
- [ ] Keep output writers independent from source adapters.
- [ ] Make internal models impossible to construct with contradictory states.
- [ ] Add deterministic sort and merge phases shared by all adapters.

Possible layout:

```text
cmd/testtranslator/
internal/results/
internal/results/junit/
internal/results/gotest/
internal/results/trx/
internal/results/nunit/
internal/results/tap/
internal/coverage/
internal/coverage/cobertura/
internal/coverage/gocover/
internal/coverage/istanbul/
internal/coverage/lcov/
internal/coverage/jacoco/
internal/coverage/opencover/
internal/diagnostics/
internal/manifest/
testdata/
```

## Reuse and licensing

- [ ] Prefer stable public Go packages when they provide the required behavior.
- [ ] Copy or port implementation code when the upstream API is internal or
      unsuitable for embedding.
- [ ] Only reuse code under licenses compatible with this repository.
- [ ] Keep an upstream record for every copied implementation:
  - repository URL;
  - commit or release;
  - copied files;
  - local modifications;
  - license;
  - required notices.
- [ ] Retain copyright headers where required.
- [ ] Add `THIRD_PARTY_NOTICES.md`.
- [ ] Add dependency and copied-code license checks to CI.
- [ ] Track upstream releases and security advisories.

Suggested copied-adapter layout:

```text
internal/results/gotest/
├── LICENSE.upstream
├── NOTICE.upstream
├── UPSTREAM.md
├── parser.go
├── parser_test.go
└── testdata/
```

## Fixtures and verification

- [ ] Build a checked-in fixture corpus for every supported format.
- [ ] Include passing, failing, skipped, timed-out, crashed, malformed, and
      empty test runs.
- [ ] Include nested suites and deeply nested subtests.
- [ ] Include Unicode and invalid-encoding edge cases where applicable.
- [ ] Include interleaved and parallel Go test events.
- [ ] Include build and discovery failures.
- [ ] Include Windows, Unix, absolute, relative, and escaping coverage paths.
- [ ] Include zero-total and partially covered files.
- [ ] Include coverage reports with and without branch/function data.
- [ ] Add golden JUnit and Cobertura outputs.
- [ ] Validate every golden output independently of the writer implementation.
- [ ] Add round-trip normalization tests for JUnit and Cobertura inputs.
- [ ] Add fuzz tests for every parser boundary.
- [ ] Add differential tests against reused upstream converters.
- [ ] Test large reports and enforce memory ceilings.
- [ ] Test output determinism across repeated runs.

## Delivery

- [ ] Initialize the Go module.
- [ ] Choose and document the project license.
- [ ] Add a concise README with supported formats and examples.
- [ ] Add architecture and adapter-authoring documentation.
- [ ] Add GitHub Actions for formatting, tests, vetting, linting, fuzz smoke
      tests, and license checks.
- [ ] Build release binaries for Linux, macOS, and Windows on amd64 and arm64.
- [ ] Publish checksums and an SBOM with every release.
- [ ] Support `go install`.
- [ ] Add a reproducible release process.
- [ ] Document compatibility guarantees for adapter names and output behavior.

## Milestones

### Milestone 1: executable foundation

- [ ] Go module and CLI skeleton.
- [ ] Adapter registry with exact format identifiers.
- [ ] Atomic output and structured diagnostics.
- [ ] JUnit and Cobertura writers plus validators.
- [ ] Fixture and golden-test harness.

### Milestone 2: Go end to end

- [ ] Go `test2json` to JUnit using gotestsum behavior.
- [ ] Go coverprofile to Cobertura using the Faktorial adapter as the seed.
- [ ] Handle subtests, interleaving, package build failures, unsafe paths, and
      statement-vs-line semantics correctly.

### Milestone 3: broad interchange formats

- [ ] TRX, NUnit, xUnit.net, Surefire/JUnit, and TAP test-result adapters.
- [ ] LCOV, Cobertura, JaCoCo, OpenCover, and Istanbul coverage adapters.

### Milestone 4: ecosystem-native formats

- [ ] Jest/Vitest, RSpec, PHPUnit, GoogleTest, CTest, Rust, Bazel, and Dart test
      adapters.
- [ ] coverage.py, Coverlet, LLVM, Clover, SimpleCov, and gcov coverage
      adapters.

### Milestone 5: production hardening

- [ ] Large-file streaming and resource limits.
- [ ] Complete third-party notices and license automation.
- [ ] Cross-platform release pipeline.
- [ ] Compatibility test matrix against major CI consumers.
- [ ] Stable v1 CLI and adapter contracts.

## Definition of done for an adapter

An adapter is not considered supported until all of the following are true:

- [ ] Its exact source format and supported versions are documented.
- [ ] It is selected explicitly through a stable format identifier.
- [ ] Valid representative fixtures convert successfully.
- [ ] Malformed and mismatched inputs fail clearly.
- [ ] Source-specific edge cases have tests.
- [ ] Conversion losses are reported.
- [ ] Output is deterministic and validates successfully.
- [ ] Path handling is safe where paths are present.
- [ ] License and upstream provenance are recorded for reused code.
- [ ] The public supported-format list includes it.

