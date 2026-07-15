// Package cobertura writes and validates Cobertura XML, the public coverage
// output contract. Output is deterministic: rates use fixed precision, files
// group into packages by directory, and no wall-clock timestamp is embedded.
package cobertura

import (
	"encoding/xml"
	"fmt"
	"path"
	"sort"
	"strconv"

	"github.com/asynkron/Asynkron.TestTranslator/internal/coverage"
)

const xmlHeader = `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
	`<!DOCTYPE coverage SYSTEM "http://cobertura.sourceforge.net/xml/coverage-04.dtd">` + "\n"

// version labels the generator; kept stable for deterministic output.
const version = "testtranslator-1.0"

type xmlCoverage struct {
	XMLName         xml.Name    `xml:"coverage"`
	LineRate        string      `xml:"line-rate,attr"`
	BranchRate      string      `xml:"branch-rate,attr"`
	LinesCovered    int64       `xml:"lines-covered,attr"`
	LinesValid      int64       `xml:"lines-valid,attr"`
	BranchesCovered int64       `xml:"branches-covered,attr"`
	BranchesValid   int64       `xml:"branches-valid,attr"`
	Complexity      string      `xml:"complexity,attr"`
	Version         string      `xml:"version,attr"`
	Timestamp       string      `xml:"timestamp,attr"`
	Sources         xmlSources  `xml:"sources"`
	Packages        xmlPackages `xml:"packages"`
}

type xmlSources struct {
	Source []string `xml:"source"`
}

type xmlPackages struct {
	Package []xmlPackage `xml:"package"`
}

type xmlPackage struct {
	Name       string     `xml:"name,attr"`
	LineRate   string     `xml:"line-rate,attr"`
	BranchRate string     `xml:"branch-rate,attr"`
	Complexity string     `xml:"complexity,attr"`
	Classes    xmlClasses `xml:"classes"`
}

type xmlClasses struct {
	Class []xmlClass `xml:"class"`
}

type xmlClass struct {
	Name       string     `xml:"name,attr"`
	Filename   string     `xml:"filename,attr"`
	LineRate   string     `xml:"line-rate,attr"`
	BranchRate string     `xml:"branch-rate,attr"`
	Complexity string     `xml:"complexity,attr"`
	Methods    xmlMethods `xml:"methods"`
	Lines      xmlLines   `xml:"lines"`
}

type xmlMethods struct {
	Method []xmlMethod `xml:"method"`
}

type xmlMethod struct {
	Name       string   `xml:"name,attr"`
	Signature  string   `xml:"signature,attr"`
	LineRate   string   `xml:"line-rate,attr"`
	BranchRate string   `xml:"branch-rate,attr"`
	Lines      xmlLines `xml:"lines"`
}

type xmlLines struct {
	Line []xmlLine `xml:"line"`
}

type xmlLine struct {
	Number            int            `xml:"number,attr"`
	Hits              int64          `xml:"hits,attr"`
	Branch            string         `xml:"branch,attr"`
	ConditionCoverage string         `xml:"condition-coverage,attr,omitempty"`
	Conditions        *xmlConditions `xml:"conditions,omitempty"`
}

type xmlConditions struct {
	Condition []xmlCondition `xml:"condition"`
}

type xmlCondition struct {
	Number   int    `xml:"number,attr"`
	Type     string `xml:"type,attr"`
	Coverage string `xml:"coverage,attr"`
}

// rate formats a coverage ratio with fixed precision for deterministic output.
func rate(v float64) string { return strconv.FormatFloat(v, 'f', 4, 64) }

// fileRates computes line and branch rates for one file.
func fileRates(f *coverage.File) (lineRate float64, lc, lv int64, branchRate float64, bc, bv int64) {
	for _, l := range f.Lines {
		lv++
		if l.Hits > 0 {
			lc++
		}
		if l.Branch {
			bc += int64(l.BranchesCovered)
			bv += int64(l.BranchesTotal)
		}
	}
	if lv > 0 {
		lineRate = float64(lc) / float64(lv)
	}
	if bv > 0 {
		branchRate = float64(bc) / float64(bv)
	}
	return
}

// Marshal renders a coverage Report as Cobertura XML. The report must already
// be normalized (see coverage.Report.Normalize).
func Marshal(r *coverage.Report) ([]byte, error) {
	doc := xmlCoverage{
		Version:    version,
		Timestamp:  "0",
		Complexity: "0",
	}
	if len(r.SourceRoots) > 0 {
		doc.Sources.Source = append([]string(nil), r.SourceRoots...)
	} else {
		doc.Sources.Source = []string{"."}
	}

	// Group files into packages by directory for deterministic output.
	byPkg := map[string][]*coverage.File{}
	for i := range r.Files {
		f := &r.Files[i]
		dir := path.Dir(f.Path)
		if dir == "." || dir == "/" {
			dir = "."
		}
		byPkg[dir] = append(byPkg[dir], f)
	}
	pkgNames := make([]string, 0, len(byPkg))
	for name := range byPkg {
		pkgNames = append(pkgNames, name)
	}
	sort.Strings(pkgNames)

	var totalLC, totalLV, totalBC, totalBV int64
	for _, pkgName := range pkgNames {
		files := byPkg[pkgName]
		sort.Slice(files, func(a, b int) bool { return files[a].Path < files[b].Path })
		pkg := xmlPackage{Name: pkgName, Complexity: "0"}
		var pLC, pLV, pBC, pBV int64
		for _, f := range files {
			lr, lc, lv, br, bc, bv := fileRates(f)
			pLC += lc
			pLV += lv
			pBC += bc
			pBV += bv
			cls := xmlClass{
				Name:       f.Path,
				Filename:   f.Path,
				LineRate:   rate(lr),
				BranchRate: rate(br),
				Complexity: "0",
			}
			for _, l := range f.Lines {
				xl := xmlLine{Number: l.Number, Hits: l.Hits, Branch: strconv.FormatBool(l.Branch)}
				if l.Branch && l.BranchesTotal > 0 {
					pct := 100 * l.BranchesCovered / l.BranchesTotal
					xl.ConditionCoverage = fmt.Sprintf("%d%% (%d/%d)", pct, l.BranchesCovered, l.BranchesTotal)
					conds := &xmlConditions{}
					for i := 0; i < l.BranchesTotal; i++ {
						cov := "0%"
						if i < l.BranchesCovered {
							cov = "100%"
						}
						conds.Condition = append(conds.Condition, xmlCondition{Number: i, Type: "jump", Coverage: cov})
					}
					xl.Conditions = conds
				}
				cls.Lines.Line = append(cls.Lines.Line, xl)
			}
			for _, fn := range f.Functions {
				cls.Methods.Method = append(cls.Methods.Method, xmlMethod{
					Name:       fn.Name,
					Signature:  "",
					LineRate:   rate(boolRate(fn.Hits > 0)),
					BranchRate: "0.0000",
				})
			}
			pkg.Classes.Class = append(pkg.Classes.Class, cls)
		}
		if pLV > 0 {
			pkg.LineRate = rate(float64(pLC) / float64(pLV))
		} else {
			pkg.LineRate = rate(0)
		}
		if pBV > 0 {
			pkg.BranchRate = rate(float64(pBC) / float64(pBV))
		} else {
			pkg.BranchRate = rate(0)
		}
		doc.Packages.Package = append(doc.Packages.Package, pkg)
		totalLC += pLC
		totalLV += pLV
		totalBC += pBC
		totalBV += pBV
	}

	doc.LinesCovered = totalLC
	doc.LinesValid = totalLV
	doc.BranchesCovered = totalBC
	doc.BranchesValid = totalBV
	if totalLV > 0 {
		doc.LineRate = rate(float64(totalLC) / float64(totalLV))
	} else {
		doc.LineRate = rate(0)
	}
	if totalBV > 0 {
		doc.BranchRate = rate(float64(totalBC) / float64(totalBV))
	} else {
		doc.BranchRate = rate(0)
	}

	body, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal cobertura: %w", err)
	}
	out := make([]byte, 0, len(xmlHeader)+len(body)+1)
	out = append(out, xmlHeader...)
	out = append(out, body...)
	out = append(out, '\n')
	return out, nil
}

func boolRate(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
