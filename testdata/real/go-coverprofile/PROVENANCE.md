# go-coverprofile fixtures

Real Go coverage profiles produced by the Go toolchain (`go test
-coverprofile`) against this repository's own `internal/pathutil` package. They
are genuine tool output, not hand-written.

| File | Command |
|------|---------|
| `pathutil-set.out`    | `go test -covermode=set    -coverprofile=pathutil-set.out    ./internal/pathutil/` |
| `pathutil-count.out`  | `go test -covermode=count  -coverprofile=pathutil-count.out  ./internal/pathutil/` |
| `pathutil-atomic.out` | `go test -covermode=atomic -coverprofile=pathutil-atomic.out ./internal/pathutil/` |

Each file exercises one of the three Go coverage modes (`set`, `count`,
`atomic`), so the adapter is proven against all block-count variants. Paths are
Go import paths (`github.com/asynkron/testtranslator/internal/pathutil/...`);
supply `--go-module github.com/asynkron/testtranslator` (or `--repo-root`) to
reroot them to repository-relative form.

## Regenerating

From the repository root:

```sh
for m in set count atomic; do
  go test -covermode=$m -coverprofile=testdata/real/go-coverprofile/pathutil-$m.out ./internal/pathutil/
done
```

The exact numbers shift as `internal/pathutil` changes; that is expected. The
fixtures exist to exercise real coverprofile syntax and the statement-vs-line
derivation, not to pin specific coverage percentages.

## License

Generated from this repository's own source; covered by this repository's
license. No third-party code.
