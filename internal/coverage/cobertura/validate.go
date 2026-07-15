package cobertura

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
)

type vCoverage struct {
	XMLName      xml.Name   `xml:"coverage"`
	LineRate     string     `xml:"line-rate,attr"`
	BranchRate   string     `xml:"branch-rate,attr"`
	LinesCovered *int64     `xml:"lines-covered,attr"`
	LinesValid   *int64     `xml:"lines-valid,attr"`
	Packages     []vPackage `xml:"packages>package"`
}

type vPackage struct {
	Name    string   `xml:"name,attr"`
	Classes []vClass `xml:"classes>class"`
}

type vClass struct {
	Name     string  `xml:"name,attr"`
	Filename string  `xml:"filename,attr"`
	Lines    []vLine `xml:"lines>line"`
}

type vLine struct {
	Number int    `xml:"number,attr"`
	Hits   int64  `xml:"hits,attr"`
	Branch string `xml:"branch,attr"`
}

// Validate parses Cobertura XML and checks structural and semantic invariants:
// a well-formed <coverage> root, numeric rates within [0,1], positive line
// numbers, non-negative hits, and declared lines-covered/lines-valid that agree
// with the actual lines. External entities are disabled.
func Validate(r io.Reader) error {
	dec := xml.NewDecoder(r)
	dec.Strict = true
	dec.Entity = map[string]string{}

	var doc vCoverage
	if err := dec.Decode(&doc); err != nil {
		return fmt.Errorf("cobertura: malformed XML: %w", err)
	}
	if doc.XMLName.Local != "coverage" {
		return fmt.Errorf("cobertura: root element is %q, want <coverage>", doc.XMLName.Local)
	}
	if err := checkRate("line-rate", doc.LineRate); err != nil {
		return err
	}
	if err := checkRate("branch-rate", doc.BranchRate); err != nil {
		return err
	}

	var lc, lv int64
	for pi := range doc.Packages {
		p := &doc.Packages[pi]
		for ci := range p.Classes {
			c := &p.Classes[ci]
			if c.Filename == "" {
				return fmt.Errorf("cobertura: class %q has no filename", c.Name)
			}
			for li := range c.Lines {
				l := &c.Lines[li]
				if l.Number <= 0 {
					return fmt.Errorf("cobertura: file %q has non-positive line number %d", c.Filename, l.Number)
				}
				if l.Hits < 0 {
					return fmt.Errorf("cobertura: file %q line %d has negative hits", c.Filename, l.Number)
				}
				if l.Branch != "true" && l.Branch != "false" {
					return fmt.Errorf("cobertura: file %q line %d has invalid branch %q", c.Filename, l.Number, l.Branch)
				}
				lv++
				if l.Hits > 0 {
					lc++
				}
			}
		}
	}
	if doc.LinesValid != nil && *doc.LinesValid != lv {
		return fmt.Errorf("cobertura: lines-valid declared %d but found %d", *doc.LinesValid, lv)
	}
	if doc.LinesCovered != nil && *doc.LinesCovered != lc {
		return fmt.Errorf("cobertura: lines-covered declared %d but found %d", *doc.LinesCovered, lc)
	}
	return nil
}

func checkRate(label, v string) error {
	if v == "" {
		return nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fmt.Errorf("cobertura: %s %q is not numeric", label, v)
	}
	if f < 0 || f > 1.0000001 {
		return fmt.Errorf("cobertura: %s %v is outside [0,1]", label, f)
	}
	return nil
}
