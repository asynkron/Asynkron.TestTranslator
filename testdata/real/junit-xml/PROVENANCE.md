# JUnit-family XML — real third-party test fixtures

These are **unmodified, byte-exact** JUnit-style XML files taken from public,
permissively-licensed repositories. They are genuine test-runner output that the
upstream projects committed as parser test fixtures / captured CI results. They
are used here **read-only** as proving material for a converter. Nothing in this
directory has been edited.

Two source repositories, both **Apache-2.0**:

- `apache/maven-surefire` — Maven Surefire's own report-parser fixtures. Single
  `<testsuite>` root, Java stack traces, `<error>`, `<failure>`, `<flakyFailure>`
  (rerun), nested/enclosed class names.
- `EnricoMi/publish-unit-test-result-action` — real pytest `--junitxml` output
  (from the Horovod test suite) captured as action fixtures. `<testsuites>` root,
  `<skipped>`, `<failure>`, passing runs, small and large.

---

## Files

### surefire-TEST-org.apache.maven.surefire.test.FailingTest.xml
- Source repo: apache/maven-surefire — https://github.com/apache/maven-surefire
- Raw URL (pinned): https://raw.githubusercontent.com/apache/maven-surefire/561b4ca356e6fae53f0f16f4862fdd22305852fb/surefire-report-parser/src/test/resources/fixture/testsuitexmlparser/TEST-org.apache.maven.surefire.test.FailingTest.xml
- Original path: surefire-report-parser/src/test/resources/fixture/testsuitexmlparser/TEST-org.apache.maven.surefire.test.FailingTest.xml
- License: Apache-2.0
- Description: Single `<testsuite>`, 2 tests both `<failure>`, full `<properties>` block and JUnit assertion stack traces.

### surefire-TEST-surefire.MyTest.xml
- Source repo: apache/maven-surefire — https://github.com/apache/maven-surefire
- Raw URL (pinned): https://raw.githubusercontent.com/apache/maven-surefire/561b4ca356e6fae53f0f16f4862fdd22305852fb/surefire-report-parser/src/test/resources/fixture/testsuitexmlparser/TEST-surefire.MyTest.xml
- Original path: surefire-report-parser/src/test/resources/fixture/testsuitexmlparser/TEST-surefire.MyTest.xml
- License: Apache-2.0
- Description: Single `<testsuite>` with one `<testcase>` producing an `<error>` (RuntimeException) with stack trace.

### surefire-TEST-surefire.MyTest-enclosed.xml
- Source repo: apache/maven-surefire — https://github.com/apache/maven-surefire
- Raw URL (pinned): https://raw.githubusercontent.com/apache/maven-surefire/561b4ca356e6fae53f0f16f4862fdd22305852fb/surefire-report-parser/src/test/resources/fixture/testsuitexmlparser/TEST-surefire.MyTest-enclosed.xml
- Original path: surefire-report-parser/src/test/resources/fixture/testsuitexmlparser/TEST-surefire.MyTest-enclosed.xml
- License: Apache-2.0
- Description: `<error>` case where the `classname` is an enclosed/nested Java class (`surefire.MyTest$A`).

### surefire-TEST-org.acme.FlakyTest.xml
- Source repo: apache/maven-surefire — https://github.com/apache/maven-surefire
- Raw URL (pinned): https://raw.githubusercontent.com/apache/maven-surefire/561b4ca356e6fae53f0f16f4862fdd22305852fb/surefire-report-parser/src/test/resources/fixture/testsuitexmlparser/TEST-org.acme.FlakyTest.xml
- Original path: surefire-report-parser/src/test/resources/fixture/testsuitexmlparser/TEST-org.acme.FlakyTest.xml
- License: Apache-2.0
- Description: Surefire 3.x rerun/flaky report — testcase reports success but carries `<flakyFailure>` reruns with large stack traces (schema version 3.0.2).

### pytest-fail.xml
- Source repo: EnricoMi/publish-unit-test-result-action — https://github.com/EnricoMi/publish-unit-test-result-action
- Raw URL (pinned): https://raw.githubusercontent.com/EnricoMi/publish-unit-test-result-action/c9bbcabcaf28c5ce2a9f8b6ee91c62873071b2ce/python/test/files/junit-xml/pytest/junit.fail.xml
- Original path: python/test/files/junit-xml/pytest/junit.fail.xml
- License: Apache-2.0
- Description: Real pytest `--junitxml` output. `<testsuites>` root, 5 tests, 1 `<failure>`, 1 `<skipped>`; small.

### pytest-mpi.static.xml
- Source repo: EnricoMi/publish-unit-test-result-action — https://github.com/EnricoMi/publish-unit-test-result-action
- Raw URL (pinned): https://raw.githubusercontent.com/EnricoMi/publish-unit-test-result-action/c9bbcabcaf28c5ce2a9f8b6ee91c62873071b2ce/python/test/files/junit-xml/pytest/junit.mpi.static.xml
- Original path: python/test/files/junit-xml/pytest/junit.mpi.static.xml
- License: Apache-2.0
- Description: Real pytest output, all 24 tests passing (0 failures/errors/skipped); `file`/`line` attributes on each `<testcase>`.

### pytest-spark.integration.1.xml
- Source repo: EnricoMi/publish-unit-test-result-action — https://github.com/EnricoMi/publish-unit-test-result-action
- Raw URL (pinned): https://raw.githubusercontent.com/EnricoMi/publish-unit-test-result-action/c9bbcabcaf28c5ce2a9f8b6ee91c62873071b2ce/python/test/files/junit-xml/pytest/junit.spark.integration.1.xml
- Original path: python/test/files/junit-xml/pytest/junit.spark.integration.1.xml
- License: Apache-2.0
- Description: Larger real pytest run, 35 tests with 2 `<skipped>`; `<testsuites>` root.

---

## Notes on pinning & verification
- All raw URLs are pinned to a full commit SHA for reproducibility.
- maven-surefire files were downloaded from `master` and confirmed byte-identical
  (`cmp`) against the pinned SHA `561b4ca356e6fae53f0f16f4862fdd22305852fb`.
- pytest files were downloaded directly at the pinned SHA
  `c9bbcabcaf28c5ce2a9f8b6ee91c62873071b2ce`.
- Every download returned HTTP 200 and begins with a valid `<?xml ...?>` / JUnit
  `<testsuite(s)>` element.

## A note on testmoapp/junitxml
The prompt suggested `testmoapp/junitxml` as an MIT example source. Its example
files are hand-authored documentation samples (not real runner output), and I
could not confirm a committed LICENSE file (root `LICENSE`/`LICENSE.md` returned
404, and the repo's About panel did not expose a license). To stay within the
"real tool output + confirmed permissive license" rules, it was **not** used.
