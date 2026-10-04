# Third-party notices

## Runtime dependencies

The `cmd/testtranslator` binary uses the **Go standard library only**.
The reusable `projection/coverage` library additionally imports
`golang.org/x/tools/cover` from the version of `golang.org/x/tools` recorded in
`go.mod` (currently v0.48.0). That package is distributed under the Go Authors'
BSD-3-Clause license with an additional patent grant.

The upstream notices are retained verbatim in
[LICENSE](.github/licenses/golang.org-x-tools/LICENSE) and
[PATENTS](.github/licenses/golang.org-x-tools/PATENTS). The `licenses` CI job
checks that the command has no third-party packages, that the only reviewed
third-party library package is exactly `golang.org/x/tools/cover`, and that the
retained license and patent notices match the installed dependency. A dependency
update that changes either notice requires an explicit notice update.

## Ported or copied implementation code

None. Every adapter in this repository is an original, clean-room implementation
written against the public, documented format contracts (for example, the
`test2json` event schema at <https://pkg.go.dev/cmd/test2json>). No parser or
converter source code has been copied or vendored from another project.

Where naming and edge-case behavior follow an existing tool's conventions — for
instance, gotestsum's handling of Go subtests and package build failures — that
influence is behavioral only and involves no copied code. If implementation code
is copied or ported in the future, it must be accompanied by an `UPSTREAM.md`
recording the repository URL, commit/release, copied files, local modifications,
license, and required notices, and be listed here.

## Redistributed test fixtures

The `testdata/real/` tree contains **unmodified** test-output files committed by
public, permissively licensed upstream projects. They are redistributed here
read-only as proving material and remain under their original licenses. Each
fixture directory carries a `PROVENANCE.md` recording, per file: the source
repository, the exact commit-pinned raw URL, the original path, the SPDX license,
and a description.

| Fixture directory | Upstream source | License |
|-------------------|-----------------|---------|
| `testdata/real/go-testjson/` | gotestyourself/gotestsum | Apache-2.0 |
| `testdata/real/junit-xml/` | see PROVENANCE.md | permissive (per file) |
| `testdata/real/trx/` | see PROVENANCE.md | permissive (per file) |
| `testdata/real/nunit/` | see PROVENANCE.md | permissive (per file) |
| `testdata/real/xunit/` | see PROVENANCE.md | permissive (per file) |
| `testdata/real/tap/` | see PROVENANCE.md | permissive (per file) |
| `testdata/real/lcov/` | see PROVENANCE.md | permissive (per file) |
| `testdata/real/istanbul-json/` | see PROVENANCE.md | permissive (per file) |
| `testdata/real/jacoco/` | see PROVENANCE.md | permissive (per file) |
| `testdata/real/cobertura/` | see PROVENANCE.md | permissive (per file) |

Refer to the `PROVENANCE.md` in each directory for the authoritative,
per-file license and source record. Directories are included only when real
fixtures were located under a permissive license.
