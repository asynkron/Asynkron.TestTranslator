# Cobertura XML — real third-party test fixtures

These are **unmodified** Cobertura-format coverage XML files taken verbatim from the
committed test suites of two public, permissively-licensed projects. They are stored
here **read-only** as proving material for the converter. No file contents were edited.

Cobertura format = root `<coverage line-rate=... branch-rate=...>` with
`<sources>` and `<packages>/<package>/<classes>/<class filename=>/<lines>/<line>`.

Two distinct generators are represented so the converter is exercised against real
variation in attribute ordering, DOCTYPE presence, `<methods>`/`<conditions>` usage,
and indentation:

- **gcovr** (C/C++ gcov coverage) — emits a `<!DOCTYPE coverage SYSTEM ...>`, `condition` elements on branch lines, 2-space indentation.
- **coverage.py** (Python) — no DOCTYPE, alphabetically-ordered attributes, tab indentation, generator comments, `<methods>` elements.

## Files

### gcovr (source repo: `gcovr/gcovr`, license: BSD-3-Clause)
Pinned commit: `755fbfb26642de081bf8691ca62cd161f76c5f15`
Raw URL base: `https://raw.githubusercontent.com/gcovr/gcovr/755fbfb26642de081bf8691ca62cd161f76c5f15/`

| local filename | original path in repo | description |
|---|---|---|
| `gcovr-no-branch-fullcoverage.xml` | `gcovr/tests/no-branch/reference/clang-10/cobertura.xml` | Tiny report, 100% line coverage, no branches (line-rate 1.0, 2/2 lines). |
| `gcovr-split-signature-methods.xml` | `gcovr/tests/split-signature/reference/gcc-5/cobertura.xml` | Full line coverage with partially covered branches (branch-rate 0.5); function/method rows. |
| `gcovr-exclusion-partial.xml` | `gcovr/tests/exclusion/reference/exclude-lines-by-pattern/clang-10/cobertura.xml` | Partial coverage (line-rate 0.75) with lines excluded by pattern; mix of covered/uncovered. |
| `gcovr-nested-packages.xml` | `gcovr/tests/nested/reference/standard/gcc-5/cobertura.xml` | Multiple nested packages (A, A/B, ...), partial coverage with branch conditions. |
| `gcovr-nested-linked-large.xml` | `gcovr/tests/nested/reference/linked/clang-10/cobertura.xml` | Largest sample (~12 KB); nested packages, many classes/lines, branch conditions. |

- License: BSD-3-Clause (SPDX: `BSD-3-Clause`). Copyright the gcovr authors / Sandia Corporation.
- Repo: https://github.com/gcovr/gcovr

### coverage.py (source repo: `nedbat/coveragepy`, license: Apache-2.0)
Pinned commit: `b4c1fda244a04caaa940ae77dd476a8bedfbf38e`
Raw URL base: `https://raw.githubusercontent.com/nedbat/coveragepy/b4c1fda244a04caaa940ae77dd476a8bedfbf38e/`

| local filename | original path in repo | description |
|---|---|---|
| `coveragepy-python-simple.xml` | `tests/gold/xml/x_xml/coverage.xml` | coverage.py "gold" output, line coverage only (line-rate 0.6667, 2/3 lines), no branches. |
| `coveragepy-python-branch.xml` | `tests/gold/xml/y_xml_branch/coverage.xml` | coverage.py "gold" output with branch coverage enabled (branch-rate 0.5, 1/2 branches). |

- License: Apache-2.0 (SPDX: `Apache-2.0`). Copyright Ned Batchelder and contributors.
- Repo: https://github.com/nedbat/coveragepy

## Notes
- Every file was downloaded with `curl` from the commit-pinned `raw.githubusercontent.com`
  URLs above (all HTTP 200) and confirmed byte-for-byte identical (md5) to the file at
  the pinned commit. Contents are unmodified.
- The `api.github.com` API was not used (blocked); discovery was done via shallow
  `git clone` of each repo at the pinned commit.
