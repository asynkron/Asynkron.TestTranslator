package coverage

import "testing"

func TestAddLineMergesDuplicates(t *testing.T) {
	f := &File{Path: "a.go"}
	if err := f.AddLine(LineHit{Number: 1, Hits: 1}); err != nil {
		t.Fatal(err)
	}
	if err := f.AddLine(LineHit{Number: 1, Hits: 2}); err != nil {
		t.Fatal(err)
	}
	if len(f.Lines) != 1 || f.Lines[0].Hits != 3 {
		t.Fatalf("duplicate lines not merged: %+v", f.Lines)
	}
}

func TestAddLineRejectsBad(t *testing.T) {
	f := &File{Path: "a.go"}
	if err := f.AddLine(LineHit{Number: 0, Hits: 1}); err == nil {
		t.Error("expected error for line number 0")
	}
	if err := f.AddLine(LineHit{Number: 1, Hits: -1}); err == nil {
		t.Error("expected error for negative hits")
	}
	if err := f.AddLine(LineHit{Number: 1, Branch: true, BranchesCovered: 3, BranchesTotal: 1}); err == nil {
		t.Error("expected error when covered branches exceed total")
	}
}

func TestNormalizeRejectsDuplicateFiles(t *testing.T) {
	r := &Report{Files: []File{{Path: "a.go"}, {Path: "a.go"}}}
	if err := r.Normalize(); err == nil {
		t.Fatal("expected error for duplicate file paths")
	}
}

func TestNormalizeSortsFilesAndLines(t *testing.T) {
	fb := File{Path: "b.go"}
	_ = fb.AddLine(LineHit{Number: 5, Hits: 1})
	_ = fb.AddLine(LineHit{Number: 2, Hits: 1})
	r := &Report{Files: []File{fb, {Path: "a.go"}}}
	if err := r.Normalize(); err != nil {
		t.Fatal(err)
	}
	if r.Files[0].Path != "a.go" {
		t.Errorf("files not sorted: %q first", r.Files[0].Path)
	}
	if r.Files[1].Lines[0].Number != 2 {
		t.Errorf("lines not sorted: %d first", r.Files[1].Lines[0].Number)
	}
}

func TestLineRate(t *testing.T) {
	f := File{Path: "a.go"}
	_ = f.AddLine(LineHit{Number: 1, Hits: 1})
	_ = f.AddLine(LineHit{Number: 2, Hits: 0})
	r := &Report{Files: []File{f}}
	if got := r.LineRate(); got != 0.5 {
		t.Fatalf("line rate = %v, want 0.5", got)
	}
}
