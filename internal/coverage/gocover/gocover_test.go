package gocover

import (
	"strings"
	"testing"

	"github.com/asynkron/Asynkron.TestTranslator/internal/coverage"
	"github.com/asynkron/Asynkron.TestTranslator/internal/diagnostics"
	"github.com/asynkron/Asynkron.TestTranslator/internal/pathutil"
)

const profile = `mode: set
github.com/example/project/pkg/foo.go:1.1,3.10 2 1
github.com/example/project/pkg/foo.go:5.1,5.20 1 0
github.com/example/project/pkg/bar.go:1.1,1.10 1 1
`

func parse(t *testing.T, text, repoRoot, mod string) *coverage.Report {
	t.Helper()
	diag := diagnostics.NewCollector()
	rep, err := Adapter{}.Parse(strings.NewReader(text), coverage.Options{
		SourceName: "cover.out",
		Paths:      pathutil.New(repoRoot, mod),
		Diag:       diag,
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return rep
}

func TestImportPathReroot(t *testing.T) {
	rep := parse(t, profile, "", "github.com/example/project")
	if len(rep.Files) != 2 {
		t.Fatalf("want 2 files, got %d: %+v", len(rep.Files), rep.Files)
	}
	// Files are sorted by path: pkg/bar.go then pkg/foo.go.
	if rep.Files[0].Path != "pkg/bar.go" || rep.Files[1].Path != "pkg/foo.go" {
		t.Fatalf("unexpected file paths: %q, %q", rep.Files[0].Path, rep.Files[1].Path)
	}
}

func TestStatementMetricNotMislabeled(t *testing.T) {
	rep := parse(t, profile, "", "github.com/example/project")
	var foo *coverage.File
	for i := range rep.Files {
		if rep.Files[i].Path == "pkg/foo.go" {
			foo = &rep.Files[i]
		}
	}
	if foo == nil {
		t.Fatal("pkg/foo.go missing")
	}
	stmt, ok := foo.Metrics[coverage.MetricStatements]
	if !ok {
		t.Fatal("statement metric missing")
	}
	// 2 covered statements (block 1) + 1 uncovered (block 2) = 3 total, 2 covered.
	if stmt.Covered != 2 || stmt.Total != 3 {
		t.Fatalf("statement metric wrong: covered=%d total=%d", stmt.Covered, stmt.Total)
	}
	if stmt.Derived {
		t.Error("statement metric should be native, not derived")
	}
	lines, ok := foo.Metrics[coverage.MetricLines]
	if !ok || !lines.Derived {
		t.Error("line metric should be present and marked derived")
	}
}

func TestUnknownModeFails(t *testing.T) {
	if _, err := (Adapter{}).Parse(strings.NewReader("mode: bogus\n"), coverage.Options{Paths: pathutil.New("", ""), Diag: diagnostics.NewCollector()}); err == nil {
		t.Fatal("expected error for unknown mode")
	}
}

func TestMissingModeFails(t *testing.T) {
	if _, err := (Adapter{}).Parse(strings.NewReader("github.com/x/y.go:1.1,2.2 1 1\n"), coverage.Options{Paths: pathutil.New("", "github.com/x"), Diag: diagnostics.NewCollector()}); err == nil {
		t.Fatal("expected error for missing mode directive")
	}
}

func TestAbsolutePathWithoutRootFails(t *testing.T) {
	// An absolute source path cannot be rerooted without --repo-root.
	prof := "mode: set\n/abs/outside/foo.go:1.1,2.2 1 1\n"
	_, err := (Adapter{}).Parse(strings.NewReader(prof), coverage.Options{Paths: pathutil.New("", ""), Diag: diagnostics.NewCollector()})
	if err == nil {
		t.Fatal("expected error for absolute path without repo root")
	}
}

func TestEscapingPathRejected(t *testing.T) {
	// A path that escapes the repository root must be rejected.
	prof := "mode: set\n/home/user/other/foo.go:1.1,2.2 1 1\n"
	_, err := (Adapter{}).Parse(strings.NewReader(prof), coverage.Options{Paths: pathutil.New("/home/user/repo", ""), Diag: diagnostics.NewCollector()})
	if err == nil {
		t.Fatal("expected error for path escaping repo root")
	}
}

func TestWithoutModuleImportPathIsLiteral(t *testing.T) {
	// Without --go-module, an import-path-looking string is taken as a literal,
	// safe, repo-relative path — deterministic, no heuristics.
	prof := "mode: set\nexample/pkg/foo.go:1.1,2.2 1 1\n"
	rep, err := (Adapter{}).Parse(strings.NewReader(prof), coverage.Options{Paths: pathutil.New("", ""), Diag: diagnostics.NewCollector()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rep.Files) != 1 || rep.Files[0].Path != "example/pkg/foo.go" {
		t.Fatalf("unexpected files: %+v", rep.Files)
	}
}
