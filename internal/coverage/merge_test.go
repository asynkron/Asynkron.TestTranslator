package coverage

import "testing"

// Union-merging the same file from two runs must treat branch and metric totals
// as structural (max, not sum) so the denominator is not doubled, while line
// execution counts are summed. Regression: totals used to be summed, doubling
// branch/statement denominators and duplicating Cobertura conditions.
func TestMergeUnionSameFileBranches(t *testing.T) {
	mk := func(covered int) *Report {
		return &Report{
			SourceRoots: []string{"."},
			Files: []File{{
				Path: "src/a.go",
				Lines: []LineHit{
					{Number: 10, Hits: 1, Branch: true, BranchesCovered: covered, BranchesTotal: 2},
				},
				Metrics: map[Metric]Count{
					MetricBranches: {Covered: int64(covered), Total: 2},
				},
			}},
		}
	}

	out, err := Merge([]*Report{mk(1), mk(1)}, "union", nil)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if len(out.Files) != 1 {
		t.Fatalf("want 1 merged file, got %d", len(out.Files))
	}
	line := out.Files[0].Lines[0]
	if line.BranchesTotal != 2 {
		t.Errorf("branch total = %d, want 2 (structural max, not summed to 4)", line.BranchesTotal)
	}
	if line.BranchesCovered > line.BranchesTotal {
		t.Errorf("branch covered %d exceeds total %d", line.BranchesCovered, line.BranchesTotal)
	}
	if line.Hits != 2 {
		t.Errorf("line hits = %d, want 2 (execution counts summed)", line.Hits)
	}
	if br := out.Files[0].Metrics[MetricBranches]; br.Total != 2 {
		t.Errorf("branch metric total = %d, want 2 (not summed to 4)", br.Total)
	}
}

// Mode "none" (the default) must refuse to combine a file that appears twice
// rather than silently merging.
func TestMergeNoneRejectsOverlap(t *testing.T) {
	r := func() *Report {
		return &Report{SourceRoots: []string{"."}, Files: []File{{Path: "src/a.go", Lines: []LineHit{{Number: 1, Hits: 1}}}}}
	}
	if _, err := Merge([]*Report{r(), r()}, "none", nil); err == nil {
		t.Fatalf("expected an error when the same file appears in multiple inputs without --merge union")
	}
}
