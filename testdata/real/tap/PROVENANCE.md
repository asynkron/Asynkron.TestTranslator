# TAP (Test Anything Protocol) — Real Third-Party Fixtures

These are **unmodified** third-party TAP output fixtures, committed as test
fixtures/examples in public GitHub repositories. They are used **read-only** as
proving material for a converter. No file contents have been altered.

Two upstream sources:

1. **tapjs/tapjs** (the `tap-parser` package, `node-tap`) — a large corpus of
   raw `.tap` streams that the parser is tested against. Package path
   `src/parser/`, licensed **MIT** (see `src/parser/LICENSE`:
   "This software is released under the MIT license: Copyright (c) James
   Halliday and Isaac Z. Schlueter"). Files pinned to commit
   `35a81cb16e545d2ae26e92146dfb7fa6d2a897ef`.
2. **python-tap/tappy** — the project's sample TAP output file, licensed
   **BSD-2-Clause** (Copyright (c) 2019, Matt Layman and contributors).
   Pinned to commit `ababbdc90cdac23d8137017aa6ce282c5216e7e9`.

---

## Files

### tapjs_basic.tap
- Source repo: tapjs/tapjs — https://github.com/tapjs/tapjs
- Raw URL: https://raw.githubusercontent.com/tapjs/tapjs/35a81cb16e545d2ae26e92146dfb7fa6d2a897ef/src/parser/test/fixtures/basic.tap
- Original path: src/parser/test/fixtures/basic.tap
- License: MIT
- Description: Minimal stream with a `1..6` plan and bare `ok`/`not ok` lines (no test numbers).

### tapjs_not-ok.tap
- Source repo: tapjs/tapjs — https://github.com/tapjs/tapjs
- Raw URL: https://raw.githubusercontent.com/tapjs/tapjs/35a81cb16e545d2ae26e92146dfb7fa6d2a897ef/src/parser/test/fixtures/not-ok.tap
- Original path: src/parser/test/fixtures/not-ok.tap
- License: MIT
- Description: `TAP version 13` stream with passing and failing assertions, comment lines, and a trailing pass/fail summary.

### tapjs_not-ok-todo.tap
- Source repo: tapjs/tapjs — https://github.com/tapjs/tapjs
- Raw URL: https://raw.githubusercontent.com/tapjs/tapjs/35a81cb16e545d2ae26e92146dfb7fa6d2a897ef/src/parser/test/fixtures/not-ok-todo.tap
- Original path: src/parser/test/fixtures/not-ok-todo.tap
- License: MIT
- Description: Test::More-style output where a failing assertion carries a `# TODO` directive.

### tapjs_simple_yaml.tap
- Source repo: tapjs/tapjs — https://github.com/tapjs/tapjs
- Raw URL: https://raw.githubusercontent.com/tapjs/tapjs/35a81cb16e545d2ae26e92146dfb7fa6d2a897ef/src/parser/test/fixtures/simple_yaml.tap
- Original path: src/parser/test/fixtures/simple_yaml.tap
- License: MIT
- Description: `TAP version 13` stream with indented YAML (`---`/`...`) diagnostic blocks attached to passing assertions.

### tapjs_skip-all-nonempty.tap
- Source repo: tapjs/tapjs — https://github.com/tapjs/tapjs
- Raw URL: https://raw.githubusercontent.com/tapjs/tapjs/35a81cb16e545d2ae26e92146dfb7fa6d2a897ef/src/parser/test/fixtures/skip-all-nonempty.tap
- Original path: src/parser/test/fixtures/skip-all-nonempty.tap
- License: MIT
- Description: Plan-level `1..1 # SKIP ...` (skip-all) directive followed by a single ok line.

### tapjs_bailout.tap
- Source repo: tapjs/tapjs — https://github.com/tapjs/tapjs
- Raw URL: https://raw.githubusercontent.com/tapjs/tapjs/35a81cb16e545d2ae26e92146dfb7fa6d2a897ef/src/parser/test/fixtures/bailout.tap
- Original path: src/parser/test/fixtures/bailout.tap
- License: MIT
- Description: Stream interrupted mid-run by a `Bail out!` line.

### tapjs_buffered-with-diag-not-ok.tap
- Source repo: tapjs/tapjs — https://github.com/tapjs/tapjs
- Raw URL: https://raw.githubusercontent.com/tapjs/tapjs/35a81cb16e545d2ae26e92146dfb7fa6d2a897ef/src/parser/test/fixtures/buffered-with-diag-not-ok.tap
- Original path: src/parser/test/fixtures/buffered-with-diag-not-ok.tap
- License: MIT
- Description: Failing buffered subtest with a YAML diagnostic block and a nested `{ ... }` child block.

### tappy_testresults.tap
- Source repo: python-tap/tappy — https://github.com/python-tap/tappy
- Raw URL: https://raw.githubusercontent.com/python-tap/tappy/ababbdc90cdac23d8137017aa6ce282c5216e7e9/testresults.tap_example
- Original path: testresults.tap_example
- License: BSD-2-Clause
- Description: Real all-passing run of 21 tests (`1..21`) from a Django project, grouped by source file with `#` comment headers. (Saved locally with a `.tap` extension; upstream name is `testresults.tap_example`.)
