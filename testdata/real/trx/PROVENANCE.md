# TRX real-output fixtures — provenance

These are **unmodified third-party test fixtures** (Visual Studio / MSTest `.trx`
test-result files) committed in public repositories. They are vendored here
**read-only**, byte-for-byte as downloaded, solely as proving material for the
TestTranslator converter. No file contents were altered. Each file is a genuine
test-runner output committed as a sample/fixture in its source repo.

Format: root `<TestRun xmlns="http://microsoft.com/schemas/VisualStudio/TeamTest/2010">`
with `<Results><UnitTestResult outcome="...">` entries and a `<ResultSummary>`.

| local file | outcomes present | description |
|---|---|---|
| `trxfileparser_SingleTestPassed.trx` | Passed | Single passing test with StdOut output |
| `trxfileparser_FrameworkFailure.trx` | Error, Warning | Run that failed at the framework level (RunInfos with Error/Warning) |
| `trxfileparser_TestWithCases.trx` | Passed (x3) | Data-driven test with per-case `InnerResults` |
| `aquality_mstest-example.trx` | Failed (x2) | Larger real MSTest run incl. `<TestSettings>`, `<Deployment>`, ErrorInfo/StackTrace |

## Per-file details

### trxfileparser_SingleTestPassed.trx
- Source repo: `HamedFathi/TrxFileParser` — https://github.com/HamedFathi/TrxFileParser
- Raw URL (commit-pinned): https://raw.githubusercontent.com/HamedFathi/TrxFileParser/39bb872e86682bc76c10446b0767ed5ebb5ff284/TrxFileParserTests/SampleTrxFiles/SingleTestPassed.trx
- Original path: `TrxFileParserTests/SampleTrxFiles/SingleTestPassed.trx`
- License: MIT
- Description: One passing `UnitTestResult` with standard output captured.

### trxfileparser_FrameworkFailure.trx
- Source repo: `HamedFathi/TrxFileParser` — https://github.com/HamedFathi/TrxFileParser
- Raw URL (commit-pinned): https://raw.githubusercontent.com/HamedFathi/TrxFileParser/39bb872e86682bc76c10446b0767ed5ebb5ff284/TrxFileParserTests/SampleTrxFiles/FrameworkFailure.trx
- Original path: `TrxFileParserTests/SampleTrxFiles/FrameworkFailure.trx`
- License: MIT
- Description: A run reporting a test-framework-level failure via `<RunInfos>` (Error + Warning), ResultSummary outcome "Completed".

### trxfileparser_TestWithCases.trx
- Source repo: `HamedFathi/TrxFileParser` — https://github.com/HamedFathi/TrxFileParser
- Raw URL (commit-pinned): https://raw.githubusercontent.com/HamedFathi/TrxFileParser/39bb872e86682bc76c10446b0767ed5ebb5ff284/TrxFileParserTests/SampleTrxFiles/TestWithCases.trx
- Original path: `TrxFileParserTests/SampleTrxFiles/TestWithCases.trx`
- License: MIT
- Description: A data-driven test whose aggregate `UnitTestResult` contains `InnerResults` with three passing per-iteration results.

### aquality_mstest-example.trx
- Source repo: `aquality-automation/aquality-tracking-api` — https://github.com/aquality-automation/aquality-tracking-api
- Raw URL (commit-pinned): https://raw.githubusercontent.com/aquality-automation/aquality-tracking-api/7405fbb29b9a6bf2cedc6313ea8f4c1616c6e75b/src/main/webapp/doc/examples/mstest-example.trx
- Original path: `src/main/webapp/doc/examples/mstest-example.trx`
- License: Apache-2.0
- Description: A fuller real MSTest run used as a documentation/import example, including `<TestSettings>`, `<Deployment>`, and failing results with `<ErrorInfo><StackTrace>`.

## Notes
- All raw URLs are pinned to a specific commit SHA for reproducibility.
- All source repos are under permissive licenses (MIT, Apache-2.0); their license
  text remains with the upstream repositories.
- Files verified as HTTP 200, non-empty, and well-formed XML with a
  `<TestRun>` root in the TeamTest/2010 namespace at download time.
