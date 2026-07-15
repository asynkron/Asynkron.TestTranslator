// Package diagnostics collects structured conversion warnings and notes. It
// supports a human-readable rendering on stderr and a machine-readable JSON
// mode. Diagnostics never carry generated artifacts; those go to stdout or the
// requested output file.
package diagnostics

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

// Severity classifies a diagnostic.
type Severity string

const (
	// SeverityWarning marks a lossy or unsupported mapping that did not stop
	// conversion.
	SeverityWarning Severity = "warning"
	// SeverityNote marks informational output such as a derived-metric notice.
	SeverityNote Severity = "note"
)

// Diagnostic is a single structured message.
type Diagnostic struct {
	Severity Severity `json:"severity"`
	// Code is a stable, machine-readable identifier (e.g. "lossy.branches").
	Code string `json:"code"`
	// Message is the human-readable explanation.
	Message string `json:"message"`
	// Source optionally identifies the input file or adapter.
	Source string `json:"source,omitempty"`
}

// Collector accumulates diagnostics deterministically.
type Collector struct {
	items []Diagnostic
}

// NewCollector returns an empty Collector.
func NewCollector() *Collector { return &Collector{} }

// Warnf records a warning diagnostic.
func (c *Collector) Warnf(code, source, format string, args ...any) {
	c.items = append(c.items, Diagnostic{
		Severity: SeverityWarning,
		Code:     code,
		Source:   source,
		Message:  fmt.Sprintf(format, args...),
	})
}

// Notef records a note diagnostic.
func (c *Collector) Notef(code, source, format string, args ...any) {
	c.items = append(c.items, Diagnostic{
		Severity: SeverityNote,
		Code:     code,
		Source:   source,
		Message:  fmt.Sprintf(format, args...),
	})
}

// Items returns a deterministically ordered copy of the collected diagnostics.
func (c *Collector) Items() []Diagnostic {
	out := make([]Diagnostic, len(c.items))
	copy(out, c.items)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Code != out[j].Code {
			return out[i].Code < out[j].Code
		}
		if out[i].Source != out[j].Source {
			return out[i].Source < out[j].Source
		}
		return out[i].Message < out[j].Message
	})
	return out
}

// Len returns the number of collected diagnostics.
func (c *Collector) Len() int { return len(c.items) }

// WriteText renders diagnostics in human-readable form to w.
func (c *Collector) WriteText(w io.Writer) error {
	for _, d := range c.Items() {
		src := ""
		if d.Source != "" {
			src = " [" + d.Source + "]"
		}
		if _, err := fmt.Fprintf(w, "%s: %s%s: %s\n", d.Severity, d.Code, src, d.Message); err != nil {
			return err
		}
	}
	return nil
}

// WriteJSON renders diagnostics as a JSON array to w.
func (c *Collector) WriteJSON(w io.Writer) error {
	items := c.Items()
	if items == nil {
		items = []Diagnostic{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(items)
}
