# Edge-case fixtures

Unlike `testdata/real/` (unmodified third-party tool output), these fixtures are
**intentionally synthetic**. A real corpus cannot supply malformed, empty, or
deliberately Unicode-stressed inputs, because working tools do not emit broken
files — so these are hand-crafted to exercise the boundaries the real corpus
cannot.

| Directory | Purpose | Proven by |
|-----------|---------|-----------|
| `malformed/` | One truncated/invalid input per format; each must be rejected with a clean, described error (never a panic or silent success). | `TestMalformedRejectedCleanly` |
| `empty/` | Zero-test runs; each must convert to a valid empty JUnit document. | `TestEmptyRunsConvert` |
| `unicode/` | Test names and messages spanning CJK, emoji, Greek, Cyrillic, and accented Latin; must survive parse → JUnit marshal intact. | `TestUnicodePreserved` |

All three harnesses live in `internal/integration/edge_test.go`.
