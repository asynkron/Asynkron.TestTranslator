// Package jacoco parses JaCoCo XML coverage reports into the internal coverage
// model. JaCoCo is the de facto coverage tool for the JVM ecosystem (Java,
// Kotlin, Scala, Groovy); its XML report groups coverage by <package> and, under
// each package, by <sourcefile>.
//
// A coverage.File is produced for every <sourcefile>. Its path is the package
// @name joined with the sourcefile @name (POSIX join) and then normalized to a
// repo-relative path. Per-line coverage is read from the <line> elements, and
// the sourcefile-level <counter> elements are surfaced as native coverage
// metrics.
//
// JaCoCo does not record exact execution counts: a line reports missed and
// covered *instructions*, not hit counts. This adapter therefore models a line
// as covered (Hits=1) when it has at least one covered instruction and uncovered
// (Hits=0) otherwise. A diagnostic note records this lossy mapping.
//
// The parser is strict: it rejects documents whose root element is not
// <report>, refuses custom or external XML entities, tolerates (but never
// resolves) the DOCTYPE JaCoCo emits, and bounds the input size.
package jacoco

import (
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/asynkron/testtranslator/internal/coverage"
	"github.com/asynkron/testtranslator/internal/xmlguard"
)

func init() {
	coverage.Register("jacoco", "JaCoCo XML coverage report (JVM: Java/Kotlin/Scala)", Adapter{}, "jacoco-xml")
}

// defaultMaxInput caps input size when opts.MaxInputBytes is unset.
const defaultMaxInput = 256 << 20

// Adapter implements coverage.Adapter for JaCoCo XML input.
type Adapter struct{}

// xmlReport is the JaCoCo document root. XMLName is left untagged so any root
// element decodes; the element name is then verified explicitly to produce a
// clear error rather than a generic decoder failure.
type xmlReport struct {
	XMLName  xml.Name
	Name     string       `xml:"name,attr"`
	Packages []xmlPackage `xml:"package"`
}

// xmlPackage is a JaCoCo <package>; its @name is a slash-separated path such as
// "com/example".
type xmlPackage struct {
	Name        string          `xml:"name,attr"`
	Sourcefiles []xmlSourcefile `xml:"sourcefile"`
}

// xmlSourcefile is a JaCoCo <sourcefile>; @name is the bare file name (e.g.
// "Foo.java"). It carries per-line coverage and file-level counters.
type xmlSourcefile struct {
	Name     string       `xml:"name,attr"`
	Lines    []xmlLine    `xml:"line"`
	Counters []xmlCounter `xml:"counter"`
}

// xmlLine is a JaCoCo <line>. nr is the line number; ci/mi are covered/missed
// instructions; cb/mb are covered/missed branches.
type xmlLine struct {
	Nr int `xml:"nr,attr"`
	Mi int `xml:"mi,attr"`
	Ci int `xml:"ci,attr"`
	Mb int `xml:"mb,attr"`
	Cb int `xml:"cb,attr"`
}

// xmlCounter is a JaCoCo <counter>. Type is one of INSTRUCTION, BRANCH, LINE,
// METHOD, CLASS, COMPLEXITY; Missed/Covered are absolute counts.
type xmlCounter struct {
	Type    string `xml:"type,attr"`
	Missed  int64  `xml:"missed,attr"`
	Covered int64  `xml:"covered,attr"`
}

// Parse reads a JaCoCo XML document and produces a normalized coverage report.
// It errors if the root element is not <report>.
func (Adapter) Parse(r io.Reader, opts coverage.Options) (*coverage.Report, error) {
	limit := opts.MaxInputBytes
	if limit <= 0 {
		limit = defaultMaxInput
	}
	lr := &io.LimitedReader{R: r, N: limit + 1}

	// Read the (bounded) input up front so a leading DOCTYPE directive can be
	// stripped before decoding. JaCoCo reports normally reference the JaCoCo
	// DTD; the standard decoder skips a DOCTYPE fine, but stripping it first
	// guarantees the directive can never trigger entity resolution.
	raw, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("jacoco: reading input: %w", err)
	}
	if lr.N <= 0 {
		return nil, fmt.Errorf("jacoco: input exceeds %d byte limit", limit)
	}
	raw = stripDoctype(raw)
	if err := xmlguard.Check(raw, xmlguard.DefaultMaxDepth); err != nil {
		return nil, fmt.Errorf("jacoco: %w", err)
	}

	dec := xml.NewDecoder(strings.NewReader(string(raw)))
	dec.Strict = true
	// Refuse custom/external entities: an empty entity map means any &name;
	// reference that is not a predefined XML entity fails decoding.
	dec.Entity = map[string]string{}

	var doc xmlReport
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("jacoco: malformed XML: %w", err)
	}
	if doc.XMLName.Local != "report" {
		return nil, fmt.Errorf("jacoco: root element is %q, want <report>", doc.XMLName.Local)
	}

	report := &coverage.Report{
		Producer:          "jacoco",
		InterchangeFormat: "jacoco-xml",
	}

	for pi := range doc.Packages {
		pkg := &doc.Packages[pi]
		for si := range pkg.Sourcefiles {
			sf := &pkg.Sourcefiles[si]
			f, err := convertSourcefile(pkg, sf, opts)
			if err != nil {
				return nil, err
			}
			report.Files = append(report.Files, *f)
		}
	}

	if len(report.Files) == 0 {
		return nil, fmt.Errorf("jacoco: no <sourcefile> entries found")
	}

	// JaCoCo line coverage is boolean (instruction-derived), not exact counts.
	opts.Diag.Notef("jacoco.lines", opts.SourceName,
		"JaCoCo line hits are boolean-covered (derived from covered instructions), not exact execution counts")

	if err := report.Normalize(); err != nil {
		return nil, fmt.Errorf("jacoco: %w", err)
	}
	return report, nil
}

// convertSourcefile maps one JaCoCo <sourcefile> (within its <package>) to a
// coverage.File.
func convertSourcefile(pkg *xmlPackage, sf *xmlSourcefile, opts coverage.Options) (*coverage.File, error) {
	rel, err := resolvePath(opts, pkg.Name, sf.Name)
	if err != nil {
		return nil, err
	}
	f := &coverage.File{Path: rel}

	for _, l := range sf.Lines {
		hit := coverage.LineHit{Number: l.Nr}
		// JaCoCo does not expose exec counts; a line is "covered" when it has
		// at least one covered instruction.
		if l.Ci > 0 {
			hit.Hits = 1
		}
		if l.Cb+l.Mb > 0 {
			hit.Branch = true
			hit.BranchesCovered = l.Cb
			hit.BranchesTotal = l.Cb + l.Mb
		}
		if err := f.AddLine(hit); err != nil {
			return nil, fmt.Errorf("jacoco: file %s line %d: %w", rel, l.Nr, err)
		}
	}

	// Surface sourcefile-level counters as native (non-derived) metrics.
	for _, c := range sf.Counters {
		m, ok := metricFor(c.Type)
		if !ok {
			continue
		}
		f.SetMetric(m, coverage.Count{
			Covered: c.Covered,
			Total:   c.Covered + c.Missed,
		})
	}

	return f, nil
}

// metricFor maps a JaCoCo counter @type to a coverage.Metric. CLASS and
// COMPLEXITY counters have no representable equivalent and are ignored.
func metricFor(counterType string) (coverage.Metric, bool) {
	switch strings.ToUpper(strings.TrimSpace(counterType)) {
	case "INSTRUCTION":
		return coverage.MetricInstructions, true
	case "BRANCH":
		return coverage.MetricBranches, true
	case "LINE":
		return coverage.MetricLines, true
	case "METHOD":
		return coverage.MetricMethods, true
	default:
		return "", false
	}
}

// resolvePath builds the repo-relative POSIX path for a sourcefile from its
// package name and file name. When a normalizer is supplied it is used to reject
// escaping paths; otherwise the joined path is cleaned to POSIX form as-is.
func resolvePath(opts coverage.Options, pkgName, fileName string) (string, error) {
	file := strings.TrimSpace(fileName)
	if file == "" {
		return "", fmt.Errorf("jacoco: <sourcefile> missing name attribute")
	}
	joined := path.Join(strings.TrimSpace(pkgName), file)
	joined = strings.ReplaceAll(joined, "\\", "/")

	if opts.Paths != nil {
		rel, err := opts.Paths.Rel(joined)
		if err != nil {
			return "", fmt.Errorf("jacoco: cannot reroot %q: %w", joined, err)
		}
		return rel, nil
	}

	clean := path.Clean(joined)
	clean = strings.TrimPrefix(clean, "./")
	if clean == "" || clean == "." {
		return "", fmt.Errorf("jacoco: sourcefile %q resolves to empty path", joined)
	}
	return clean, nil
}

// stripDoctype removes a single leading <!DOCTYPE ...> directive (including one
// with an internal [ ... ] subset) from the head of an XML document. Leading
// whitespace and an optional XML declaration are preserved. This keeps the
// decoder from ever having to look at DTD content while still accepting the
// DOCTYPE JaCoCo emits.
func stripDoctype(data []byte) []byte {
	s := string(data)
	i := strings.Index(s, "<!DOCTYPE")
	if i < 0 {
		return data
	}
	// Only strip a DOCTYPE that appears in the prolog, i.e. before any element.
	if lt := strings.Index(s, "<"); lt >= 0 && lt < i {
		// Ensure everything before the DOCTYPE is prolog (declaration/comment
		// or whitespace). If a real element opens first, leave the input
		// untouched and let the decoder handle it.
		prefix := strings.TrimSpace(s[:i])
		if prefix != "" && !strings.HasPrefix(prefix, "<?") && !strings.HasPrefix(prefix, "<!--") {
			return data
		}
	}

	rest := s[i+len("<!DOCTYPE"):]
	depth := 0 // bracket depth for an internal subset "[ ... ]"
	for j := 0; j < len(rest); j++ {
		switch rest[j] {
		case '[':
			depth++
		case ']':
			if depth > 0 {
				depth--
			}
		case '>':
			if depth == 0 {
				return []byte(s[:i] + rest[j+1:])
			}
		}
	}
	// Unterminated DOCTYPE: leave the input as-is so the decoder reports the
	// malformed document.
	return data
}
