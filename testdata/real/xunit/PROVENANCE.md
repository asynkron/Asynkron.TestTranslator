# xUnit.net result XML — real third-party test fixtures

These files are **unmodified** third-party fixtures, committed in public repositories,
copied here read-only for testing a test-result converter. No file contents were altered.

All are genuine xUnit.net v2+ result XML documents (root `<assemblies>` with
`<assembly>` / `<collection>` / `<test result="Pass|Fail|Skip">`), used by their
source projects as real test fixtures / expected outputs (NOT XML schemas, NOT docs,
NOT hand-written toy snippets).

## Files

### From gabrielweyer/xunit-to-junit (MIT)
Repo: https://github.com/gabrielweyer/xunit-to-junit
License: MIT
Pinned commit: `c8c9d176f49944706371f3cba84288e15e668cd2`
Original path prefix: `tests/XsltTests/input/`
These are the XSLT-transformer's input fixtures: real xUnit v2 result XML covering distinct scenarios.

- **passed-test.xml**
  - Raw URL: https://raw.githubusercontent.com/gabrielweyer/xunit-to-junit/c8c9d176f49944706371f3cba84288e15e668cd2/tests/XsltTests/input/passed-test.xml
  - Original path: `tests/XsltTests/input/passed-test.xml`
  - License: MIT
  - Description: Single assembly, one test with `result="Pass"` (total=1 passed=1).

- **failed-test.xml**
  - Raw URL: https://raw.githubusercontent.com/gabrielweyer/xunit-to-junit/c8c9d176f49944706371f3cba84288e15e668cd2/tests/XsltTests/input/failed-test.xml
  - Original path: `tests/XsltTests/input/failed-test.xml`
  - License: MIT
  - Description: One failing test (`result="Fail"`) including a `<failure>` element with exception message and stack trace.

- **skipped-test.xml**
  - Raw URL: https://raw.githubusercontent.com/gabrielweyer/xunit-to-junit/c8c9d176f49944706371f3cba84288e15e668cd2/tests/XsltTests/input/skipped-test.xml
  - Original path: `tests/XsltTests/input/skipped-test.xml`
  - License: MIT
  - Description: One skipped test (`result="Skip"`) with a `<reason>` element (skipped=1).

- **inline-data-test.xml**
  - Raw URL: https://raw.githubusercontent.com/gabrielweyer/xunit-to-junit/c8c9d176f49944706371f3cba84288e15e668cd2/tests/XsltTests/input/inline-data-test.xml
  - Original path: `tests/XsltTests/input/inline-data-test.xml`
  - License: MIT
  - Description: A theory / `[InlineData]` parameterized test showing method with inline-data arguments in the test name.

- **display-name-test.xml**
  - Raw URL: https://raw.githubusercontent.com/gabrielweyer/xunit-to-junit/c8c9d176f49944706371f3cba84288e15e668cd2/tests/XsltTests/input/display-name-test.xml
  - Original path: `tests/XsltTests/input/display-name-test.xml`
  - License: MIT
  - Description: Test using a custom `DisplayName`, exercising the display-name attribute handling.

### From spekt/xunit.testlogger (MIT)
Repo: https://github.com/spekt/xunit.testlogger
License: MIT
Pinned commit: `734876d12fd38715c5b68295a0a8a3ea3c1d5942`

- **spekt-expectedLogFile.xml**
  - Raw URL: https://raw.githubusercontent.com/spekt/xunit.testlogger/734876d12fd38715c5b68295a0a8a3ea3c1d5942/test/assets/Xunit.Xml.TestLogger.NetCore.Tests/expectedLogFile.xml
  - Original path: `test/assets/Xunit.Xml.TestLogger.NetCore.Tests/expectedLogFile.xml`
  - License: MIT
  - Description: Larger real generated run — one assembly, multiple `<collection>` groups, 5 tests (2 passed, 2 failed) mixing pass/fail with failure stack traces and skip-reason output. Expected-output fixture for the xUnit vstest logger.

## Notes
- All raw URLs pinned to a commit SHA and re-downloaded to confirm byte-identical content.
- Both source repos are MIT licensed (SPDX: MIT).
- api.github.com was not used; bytes fetched via raw.githubusercontent.com.
