# go-testjson real fixtures

These are **unmodified third-party test fixtures**, committed in a public repository,
used here **read-only** as proving material for the converter. File contents are
byte-for-byte identical to the upstream source. No files were edited.

All files come from the **gotestyourself/gotestsum** repository, directory
`testjson/testdata/input/`, which contains real `go test -json` (test2json) output
captured as NDJSON — each line is a JSON object with `Action` / `Package` / `Test` /
`Output` fields.

- Source repo: gotestyourself/gotestsum — https://github.com/gotestyourself/gotestsum
- License: **Apache-2.0** (SPDX: Apache-2.0)
- Pinned commit SHA: `1eb8ea79bb9d0e58a5f0c57969cef22e5201c738`
- Raw URL base: `https://raw.githubusercontent.com/gotestyourself/gotestsum/1eb8ea79bb9d0e58a5f0c57969cef22e5201c738/testjson/testdata/input/`

| local filename | original path | raw URL | description |
|---|---|---|---|
| go-test-json.out | testjson/testdata/input/go-test-json.out | https://raw.githubusercontent.com/gotestyourself/gotestsum/1eb8ea79bb9d0e58a5f0c57969cef22e5201c738/testjson/testdata/input/go-test-json.out | Large mixed run: passing/failing/skipped tests across many internal packages, including a package that exits non-zero (badmain). |
| go-test-json-with-cover.out | testjson/testdata/input/go-test-json-with-cover.out | https://raw.githubusercontent.com/gotestyourself/gotestsum/1eb8ea79bb9d0e58a5f0c57969cef22e5201c738/testjson/testdata/input/go-test-json-with-cover.out | Run with coverage enabled; output lines include per-package coverage percentages. |
| go-test-json-with-parallel-fails.out | testjson/testdata/input/go-test-json-with-parallel-fails.out | https://raw.githubusercontent.com/gotestyourself/gotestsum/1eb8ea79bb9d0e58a5f0c57969cef22e5201c738/testjson/testdata/input/go-test-json-with-parallel-fails.out | Parallel subtests with mixed pass/fail results. |
| go-test-json-with-shuffle.out | testjson/testdata/input/go-test-json-with-shuffle.out | https://raw.githubusercontent.com/gotestyourself/gotestsum/1eb8ea79bb9d0e58a5f0c57969cef22e5201c738/testjson/testdata/input/go-test-json-with-shuffle.out | Large run executed with -shuffle (randomized order), plus a package that exits non-zero. |
| go-test-json-with-attributes.out | testjson/testdata/input/go-test-json-with-attributes.out | https://raw.githubusercontent.com/gotestyourself/gotestsum/1eb8ea79bb9d0e58a5f0c57969cef22e5201c738/testjson/testdata/input/go-test-json-with-attributes.out | Small run using newer test attributes/metadata fields. |
| go-test-json-missing-test-fail.out | testjson/testdata/input/go-test-json-missing-test-fail.out | https://raw.githubusercontent.com/gotestyourself/gotestsum/1eb8ea79bb9d0e58a5f0c57969cef22e5201c738/testjson/testdata/input/go-test-json-missing-test-fail.out | Edge case: a failing test whose fail event is missing/misordered in the stream. |
| go-test-json-misattributed.out | testjson/testdata/input/go-test-json-misattributed.out | https://raw.githubusercontent.com/gotestyourself/gotestsum/1eb8ea79bb9d0e58a5f0c57969cef22e5201c738/testjson/testdata/input/go-test-json-misattributed.out | Edge case: subtest output that gets misattributed to the wrong parent test. |
