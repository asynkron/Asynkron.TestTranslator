package coverage

import (
	"fmt"

	"github.com/asynkron/testtranslator/internal/diagnostics"
)

// Merge combines multiple coverage reports under an explicit merge mode. Mode
// "none" rejects any file that appears in more than one report; mode "union"
// sums line hits and metric counts for repeated files. Silent combination is
// never performed.
func Merge(reports []*Report, mode string, diag *diagnostics.Collector) (*Report, error) {
	if len(reports) == 0 {
		return &Report{SourceRoots: []string{"."}}, nil
	}
	if len(reports) == 1 {
		if err := reports[0].Normalize(); err != nil {
			return nil, err
		}
		return reports[0], nil
	}

	out := &Report{}
	rootSet := map[string]bool{}
	fileIdx := map[string]int{}
	for _, r := range reports {
		for _, root := range r.SourceRoots {
			if !rootSet[root] {
				rootSet[root] = true
				out.SourceRoots = append(out.SourceRoots, root)
			}
		}
		for i := range r.Files {
			f := r.Files[i]
			if idx, exists := fileIdx[f.Path]; exists {
				if mode != "union" {
					return nil, fmt.Errorf("coverage: file %q appears in multiple inputs; pass --merge union to combine them", f.Path)
				}
				if err := mergeFile(&out.Files[idx], &f); err != nil {
					return nil, err
				}
				if diag != nil {
					diag.Notef("coverage.merge", "", "unioned repeated coverage for %q", f.Path)
				}
				continue
			}
			fileIdx[f.Path] = len(out.Files)
			out.Files = append(out.Files, f)
		}
	}
	if len(out.SourceRoots) == 0 {
		out.SourceRoots = []string{"."}
	}
	if err := out.Normalize(); err != nil {
		return nil, err
	}
	return out, nil
}

// mergeFile unions src into dst: line hits are summed, branch counts combined,
// and metric counts added. Derived flags are preserved when both agree.
func mergeFile(dst, src *File) error {
	lineByNum := map[int]int{}
	for i := range dst.Lines {
		lineByNum[dst.Lines[i].Number] = i
	}
	for _, sl := range src.Lines {
		if idx, ok := lineByNum[sl.Number]; ok {
			dst.Lines[idx].Hits += sl.Hits
			if sl.Branch {
				dst.Lines[idx].Branch = true
				dst.Lines[idx].BranchesCovered += sl.BranchesCovered
				dst.Lines[idx].BranchesTotal += sl.BranchesTotal
			}
		} else {
			dst.Lines = append(dst.Lines, sl)
			lineByNum[sl.Number] = len(dst.Lines) - 1
		}
	}
	if dst.Metrics == nil {
		dst.Metrics = map[Metric]Count{}
	}
	for m, sc := range src.Metrics {
		dc, ok := dst.Metrics[m]
		if !ok {
			dst.Metrics[m] = sc
			continue
		}
		dc.Covered += sc.Covered
		dc.Total += sc.Total
		dc.Derived = dc.Derived || sc.Derived
		dst.Metrics[m] = dc
	}
	return nil
}
