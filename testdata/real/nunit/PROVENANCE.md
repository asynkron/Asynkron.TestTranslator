# NUnit result XML — real third-party fixtures

These are unmodified, byte-exact test-runner output files committed in public
GitHub repositories. They are vendored here read-only as proving material for the
converter. No file contents were altered. Each entry lists source repo, exact
pinned raw URL (commit SHA), original path, SPDX license, and a one-line
description.

Two NUnit result dialects are represented:
- NUnit 2 — root element `<test-results>`
- NUnit 3 — root element `<test-run>`

---

## nunit2_mock-assembly_TestResult.xml
- Source repo: nunit/docs — https://github.com/nunit/docs
- Raw URL (pinned): https://raw.githubusercontent.com/nunit/docs/0038eb673c7b09c69bb91e200b4580eb50f03285/docs/files/TestResult.xml
- Original path: docs/files/TestResult.xml
- License: MIT
- Description: NUnit 2.6 result for the canonical `mock-assembly.dll` sample; a failing run (total=21, 1 error, 1 failure, 4 ignored, 1 inconclusive, 3 invalid) — spans passed/failed/ignored/inconclusive, nested Assembly/Namespace/TestFixture suites.

## nunit2_teamcity_OnKeyOCL_TestResult.xml
- Source repo: JetBrains/teamcity-xml-tests-reporting — https://github.com/JetBrains/teamcity-xml-tests-reporting
- Raw URL (pinned): https://raw.githubusercontent.com/JetBrains/teamcity-xml-tests-reporting/7994d5d3394e8ad26095597c3a2e4cbe63daf3a0/tests/testData/nunit/Pragma.OnKey5.Tests.OCL.dll.TestResult.xml
- Original path: tests/testData/nunit/Pragma.OnKey5.Tests.OCL.dll.TestResult.xml
- License: Apache-2.0
- Description: NUnit 2 result committed as a parser test fixture; a passing suite (total=2) with 11 ignored/not-run tests — exercises the not-run/ignored/reason paths.

## nunit3_example.xml
- Source repo: rjtngit/nunit-html-action — https://github.com/rjtngit/nunit-html-action
- Raw URL (pinned): https://raw.githubusercontent.com/rjtngit/nunit-html-action/d4ed966158a703bb3bd0bf097c159b9c54b5b79e/example.xml
- Original path: example.xml
- License: MIT
- Description: NUnit 3 `<test-run>` example (engine 3.5.0.0), result=Failed(Child), 5 tests, 4 passed / 1 failed — small mixed pass/fail run with a failure message + stack-trace element.

## nunit3_unity_playmode-results.xml
- Source repo: game-ci/unity-test-runner — https://github.com/game-ci/unity-test-runner
- Raw URL (pinned): https://raw.githubusercontent.com/game-ci/unity-test-runner/08fd329f00a18efa297140b14ac28ebce742759e/artifacts/playmode-results.xml
- Original path: artifacts/playmode-results.xml
- License: MIT
- Description: NUnit 3 `<test-run>` produced by the Unity Test Framework (playmode), 8 tests, 2 passed / 4 failed / 2 skipped — good variety with skipped tests, reasons, and nested test suites.

---

Note on pinning: all four URLs are pinned to a specific commit SHA for
reproducibility. Files were downloaded with
`curl -sS --max-time 30 -o <dest> -w "%{http_code}" <rawurl>` and kept only on
HTTP 200 with a matching root element (`<test-results>` for NUnit 2,
`<test-run>` for NUnit 3).
