// Package coberturain parses existing Cobertura XML coverage reports into the
// internal coverage model. It accepts the standard Cobertura schema as emitted
// by tools such as coverage.py (its cobertura-style output), cobertura itself,
// and the many reporters that speak the same dialect.
//
// Each <class> becomes a coverage.File keyed by its @filename attribute. Line
// hits and branch coverage are read from the per-class <lines> block, and
// <method> elements are preserved as coverage.Function entries. The parser is
// strict: it rejects documents whose root element is not <coverage>, refuses
// custom or external XML entities, and bounds the input size.
package coberturain

import (
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"

	"github.com/asynkron/testtranslator/internal/coverage"
)

func init() {
	coverage.Register("cobertura", "Cobertura XML coverage report (incl. coverage.py cobertura output)", Adapter{}, "cobertura-xml", "coveragepy-cobertura")
}

// defaultMaxInput caps input size when opts.MaxInputBytes is unset.
const defaultMaxInput = 256 << 20

// Adapter implements coverage.Adapter for Cobertura XML input.
type Adapter struct{}

// xmlCoverage is the Cobertura document root. XMLName is left untagged so any
// root element decodes; the element name is then verified explicitly to produce
// a clear error rather than a generic decoder failure.
type xmlCoverage struct {
	XMLName  xml.Name
	Sources  xmlSources  `xml:"sources"`
	Packages xmlPackages `xml:"packages"`
}

type xmlSources struct {
	Source []string `xml:"source"`
}

type xmlPackages struct {
	Package []xmlPackage `xml:"package"`
}

type xmlPackage struct {
	Name    string     `xml:"name,attr"`
	Classes xmlClasses `xml:"classes"`
}

type xmlClasses struct {
	Class []xmlClass `xml:"class"`
}

type xmlClass struct {
	Name     string     `xml:"name,attr"`
	Filename string     `xml:"filename,attr"`
	Methods  xmlMethods `xml:"methods"`
	Lines    xmlLines   `xml:"lines"`
}

type xmlMethods struct {
	Method []xmlMethod `xml:"method"`
}

type xmlMethod struct {
	Name      string   `xml:"name,attr"`
	Signature string   `xml:"signature,attr"`
	LineRate  string   `xml:"line-rate,attr"`
	Lines     xmlLines `xml:"lines"`
}

type xmlLines struct {
	Line []xmlLine `xml:"line"`
}

type xmlLine struct {
	Number            int    `xml:"number,attr"`
	Hits              int64  `xml:"hits,attr"`
	Branch            string `xml:"branch,attr"`
	ConditionCoverage string `xml:"condition-coverage,attr"`
}

// Parse reads a Cobertura XML document and produces a normalized coverage
// report. It errors if the root element is not <coverage>.
func (Adapter) Parse(r io.Reader, opts coverage.Options) (*coverage.Report, error) {
	limit := opts.MaxInputBytes
	if limit <= 0 {
		limit = defaultMaxInput
	}
	lr := &io.LimitedReader{R: r, N: limit + 1}
	dec := xml.NewDecoder(lr)
	dec.Strict = true
	dec.Entity = map[string]string{}

	var doc xmlCoverage
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("coberturain: malformed XML: %w", err)
	}
	if lr.N <= 0 {
		return nil, fmt.Errorf("coberturain: input exceeds %d byte limit", limit)
	}
	if doc.XMLName.Local != "coverage" {
		return nil, fmt.Errorf("coberturain: root element is %q, want <coverage>", doc.XMLName.Local)
	}

	report := &coverage.Report{
		Producer:          "cobertura",
		InterchangeFormat: "cobertura-xml",
	}
	for _, s := range doc.Sources.Source {
		if s = strings.TrimSpace(s); s != "" {
			report.SourceRoots = append(report.SourceRoots, s)
		}
	}

	for pi := range doc.Packages.Package {
		pkg := &doc.Packages.Package[pi]
		for ci := range pkg.Classes.Class {
			cls := &pkg.Classes.Class[ci]
			f, err := convertClass(cls, opts)
			if err != nil {
				return nil, err
			}
			report.Files = append(report.Files, *f)
		}
	}

	if len(report.Files) == 0 {
		return nil, fmt.Errorf("coberturain: no <class> entries found")
	}

	if err := report.Normalize(); err != nil {
		return nil, fmt.Errorf("coberturain: %w", err)
	}
	return report, nil
}

// convertClass maps one Cobertura <class> to a coverage.File.
func convertClass(cls *xmlClass, opts coverage.Options) (*coverage.File, error) {
	rel, err := resolvePath(opts, cls.Filename)
	if err != nil {
		return nil, err
	}
	f := &coverage.File{Path: rel}

	for _, l := range cls.Lines.Line {
		hit := coverage.LineHit{
			Number: l.Number,
			Hits:   l.Hits,
			Branch: strings.EqualFold(strings.TrimSpace(l.Branch), "true"),
		}
		if hit.Branch && l.ConditionCoverage != "" {
			covered, total, ok := parseConditionCoverage(l.ConditionCoverage)
			if ok {
				hit.BranchesCovered = covered
				hit.BranchesTotal = total
			} else {
				opts.Diag.Warnf("coberturain.condition", opts.SourceName,
					"file %s line %d: unparseable condition-coverage %q; branch totals dropped",
					rel, l.Number, l.ConditionCoverage)
			}
		}
		if err := f.AddLine(hit); err != nil {
			return nil, fmt.Errorf("coberturain: file %s line %d: %w", rel, l.Number, err)
		}
	}

	for _, m := range cls.Methods.Method {
		fn := coverage.Function{Name: m.Name}
		if m.Signature != "" {
			fn.Name = m.Name + m.Signature
		}
		fn.Line = firstMethodLine(&m)
		fn.Hits = methodHits(&m)
		f.Functions = append(f.Functions, fn)
	}

	return f, nil
}

// parseConditionCoverage extracts the covered/total counts from a Cobertura
// condition-coverage attribute such as "50% (1/2)". It returns ok=false when the
// "(covered/total)" fraction is missing or malformed.
func parseConditionCoverage(s string) (covered, total int, ok bool) {
	open := strings.IndexByte(s, '(')
	closeIdx := strings.IndexByte(s, ')')
	if open < 0 || closeIdx < 0 || closeIdx < open {
		return 0, 0, false
	}
	inner := s[open+1 : closeIdx]
	slash := strings.IndexByte(inner, '/')
	if slash < 0 {
		return 0, 0, false
	}
	c, err1 := strconv.Atoi(strings.TrimSpace(inner[:slash]))
	t, err2 := strconv.Atoi(strings.TrimSpace(inner[slash+1:]))
	if err1 != nil || err2 != nil || c < 0 || t < 0 || c > t {
		return 0, 0, false
	}
	return c, t, true
}

// firstMethodLine returns the smallest line number declared inside a method's
// <lines> block, or 0 when the method carries no lines.
func firstMethodLine(m *xmlMethod) int {
	first := 0
	for _, l := range m.Lines.Line {
		if l.Number <= 0 {
			continue
		}
		if first == 0 || l.Number < first {
			first = l.Number
		}
	}
	return first
}

// methodHits reports a positive hit count when the method appears to have
// executed: either its line-rate is above zero or any of its lines was hit.
func methodHits(m *xmlMethod) int64 {
	if r, err := strconv.ParseFloat(strings.TrimSpace(m.LineRate), 64); err == nil && r > 0 {
		return 1
	}
	for _, l := range m.Lines.Line {
		if l.Hits > 0 {
			return 1
		}
	}
	return 0
}

// resolvePath normalizes a Cobertura @filename into a repo-relative POSIX path.
// When a normalizer is supplied it is used to reject escaping paths; otherwise
// the filename is cleaned to POSIX form as-is (Cobertura filenames are already
// source-root relative).
func resolvePath(opts coverage.Options, filename string) (string, error) {
	name := strings.TrimSpace(filename)
	if name == "" {
		return "", fmt.Errorf("coberturain: <class> missing filename attribute")
	}
	if opts.Paths != nil {
		rel, err := opts.Paths.Rel(name)
		if err != nil {
			return "", fmt.Errorf("coberturain: cannot reroot %q: %w", name, err)
		}
		return rel, nil
	}
	clean := path.Clean(strings.ReplaceAll(name, "\\", "/"))
	clean = strings.TrimPrefix(clean, "./")
	if clean == "" || clean == "." {
		return "", fmt.Errorf("coberturain: filename %q resolves to empty path", filename)
	}
	return clean, nil
}
