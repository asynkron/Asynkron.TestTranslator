# LCOV tracefile fixtures — provenance

These are **unmodified, third-party LCOV `.info` tracefiles** taken verbatim from
public GitHub repositories under permissive (OSI) licenses. They are used
**read-only** as proving material for the test-output converter. No file in this
directory has been edited; contents are byte-for-byte as committed upstream.

Each is genuine coverage-tool output committed to a public repo (either as a
test fixture in a coverage-tooling project, or as a real `coverage/lcov.info`
produced by that project's own test run). The set spans several formats: JS
(istanbul) with branch data, Windows backslash paths, C++ `Class::method`
function names, a large Dart/Flutter run with many zero-hit records, and a Rust
run with mangled symbol names.

| Local file | Repo | License (SPDX) | Description |
|---|---|---|---|
| `lcov-merger-basic-a.info` | mweibel/lcov-result-merger | MIT | JS/istanbul tracefile: `FN`/`FNDA` functions, `DA` line hits, `BRDA` branch data, two `SF` records |
| `lcov-merger-windows.info` | mweibel/lcov-result-merger | MIT | Tracefile with a Windows drive-letter/backslash `SF:` path; `DA` line-hit records only |
| `lcov-merger-fn-colon.info` | mweibel/lcov-result-merger | MIT | Tracefile whose `FN:` entries use C++ `Class::method` names (colons in function names) |
| `signals-dart.info` | rodydavis/signals.dart | Apache-2.0 | Large Dart/Flutter `coverage/lcov.info`: many `SF` records, several fully zero-hit (`LF:0`/`LH:0`) files |
| `stralg-rust.info` | mailund/stralg-in-rust | MIT | Rust `lcov.info`: `FN`/`FNDA` with mangled Rust symbol names, `DA` and `BRDA` records |

## Sources & exact raw URLs

### lcov-merger-basic-a.info
- Repo: https://github.com/mweibel/lcov-result-merger
- Original path: `test/fixtures/basic/a/lcov.info`
- License: MIT (repo `package.json` `"license": "MIT"`)
- Raw URL (commit-pinned):
  `https://raw.githubusercontent.com/mweibel/lcov-result-merger/e0351db6a0fca774492c7ad1ac2da32cdbedfbf4/test/fixtures/basic/a/lcov.info`

### lcov-merger-windows.info
- Repo: https://github.com/mweibel/lcov-result-merger
- Original path: `test/fixtures/windows/lcov.info`
- License: MIT
- Raw URL (commit-pinned):
  `https://raw.githubusercontent.com/mweibel/lcov-result-merger/e0351db6a0fca774492c7ad1ac2da32cdbedfbf4/test/fixtures/windows/lcov.info`

### lcov-merger-fn-colon.info
- Repo: https://github.com/mweibel/lcov-result-merger
- Original path: `test/fixtures/fn-colon/lcov.info`
- License: MIT
- Raw URL (commit-pinned):
  `https://raw.githubusercontent.com/mweibel/lcov-result-merger/e0351db6a0fca774492c7ad1ac2da32cdbedfbf4/test/fixtures/fn-colon/lcov.info`

### signals-dart.info
- Repo: https://github.com/rodydavis/signals.dart
- Original path: `lcov.info` (repo root)
- License: Apache-2.0 (repo root `LICENSE`)
- Raw URL (commit-pinned):
  `https://raw.githubusercontent.com/rodydavis/signals.dart/57a1ace989f07f6769d02a078ae23f499dc79359/lcov.info`

### stralg-rust.info
- Repo: https://github.com/mailund/stralg-in-rust
- Original path: `lcov.info` (repo root)
- License: MIT (repo root `LICENSE`)
- Raw URL (commit-pinned):
  `https://raw.githubusercontent.com/mailund/stralg-in-rust/66698ae5ba65a419fc5a8fe2998d4addff9b12ec/lcov.info`

## Notes
- The three `lcov-merger-*` files are committed test fixtures in the
  lcov-result-merger project (a coverage-tooling repo), pinned to commit
  `e0351db6a0fca774492c7ad1ac2da32cdbedfbf4`.
- `signals-dart.info` and `stralg-rust.info` are real `lcov.info` files produced
  by those projects' own coverage runs and committed to their repos, pinned to
  the commit SHAs shown above.
- The official `linux-test-project/lcov` tool repo was intentionally NOT used as
  a source: it is GPL-licensed, which is outside the permissive-license scope.
