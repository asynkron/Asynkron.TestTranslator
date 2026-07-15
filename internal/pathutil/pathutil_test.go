package pathutil

import "testing"

func TestRelRelative(t *testing.T) {
	n := New("", "")
	got, err := n.Rel("pkg/foo.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "pkg/foo.go" {
		t.Fatalf("got %q, want pkg/foo.go", got)
	}
}

func TestRelWindowsSeparators(t *testing.T) {
	n := New("", "")
	got, err := n.Rel(`pkg\sub\foo.go`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "pkg/sub/foo.go" {
		t.Fatalf("got %q, want pkg/sub/foo.go", got)
	}
}

func TestRelAbsoluteReroot(t *testing.T) {
	n := New("/home/user/repo", "")
	got, err := n.Rel("/home/user/repo/pkg/foo.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "pkg/foo.go" {
		t.Fatalf("got %q, want pkg/foo.go", got)
	}
}

func TestRelWindowsDriveReroot(t *testing.T) {
	n := New(`C:\work\repo`, "")
	got, err := n.Rel(`C:\work\repo\pkg\foo.go`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "pkg/foo.go" {
		t.Fatalf("got %q, want pkg/foo.go", got)
	}
}

func TestRelRejectsEscape(t *testing.T) {
	n := New("", "")
	if _, err := n.Rel("../outside.go"); err == nil {
		t.Fatal("expected error for escaping path")
	}
}

func TestRelRejectsOutsideRepoRoot(t *testing.T) {
	n := New("/home/user/repo", "")
	if _, err := n.Rel("/etc/passwd"); err == nil {
		t.Fatal("expected error for path outside repo root")
	}
}

func TestRelAbsoluteWithoutRoot(t *testing.T) {
	n := New("", "")
	if _, err := n.Rel("/abs/foo.go"); err == nil {
		t.Fatal("expected error for absolute path without repo root")
	}
}

func TestRelFromImportPath(t *testing.T) {
	n := New("", "github.com/example/project")
	got, err := n.RelFromImportPath("github.com/example/project/pkg/foo.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "pkg/foo.go" {
		t.Fatalf("got %q, want pkg/foo.go", got)
	}
}

func TestRelFromImportPathOutsideModule(t *testing.T) {
	n := New("", "github.com/example/project")
	if _, err := n.RelFromImportPath("github.com/other/thing/foo.go"); err == nil {
		t.Fatal("expected error for import path outside module")
	}
}

func TestRelFromImportPathRequiresModule(t *testing.T) {
	n := New("", "")
	if _, err := n.RelFromImportPath("x/y.go"); err == nil {
		t.Fatal("expected error when module path is unset")
	}
}
