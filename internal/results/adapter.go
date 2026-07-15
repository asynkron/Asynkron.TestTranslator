package results

import (
	"fmt"
	"io"
	"sort"

	"github.com/asynkron/testtranslator/internal/diagnostics"
)

// Options carries conversion configuration common to result adapters.
type Options struct {
	// SourceName labels the input in diagnostics (e.g. the file path).
	SourceName string
	// Diag collects warnings and notes. Never nil when passed by the CLI.
	Diag *diagnostics.Collector
	// MaxInputBytes bounds the number of bytes an adapter will read (0 = use
	// the adapter default).
	MaxInputBytes int64
}

// Adapter parses one exact source format into the internal Report model. An
// adapter must fail clearly on malformed or mismatched input rather than
// guessing.
type Adapter interface {
	// Parse reads the source and returns a Report. Implementations must not
	// silently discard representable information.
	Parse(r io.Reader, opts Options) (*Report, error)
}

// registration binds a canonical id and its aliases to an adapter and a short
// description.
type registration struct {
	id          string
	aliases     []string
	description string
	adapter     Adapter
}

// registry maps every canonical id and alias to a registration.
var registry = map[string]*registration{}

// order preserves canonical ids in registration order for stable listing.
var order []string

// Register adds an adapter under a canonical id plus optional aliases. It
// panics on duplicate identifiers because registration happens at init time and
// a collision is a programming error.
func Register(id, description string, adapter Adapter, aliases ...string) {
	reg := &registration{id: id, aliases: aliases, description: description, adapter: adapter}
	if _, exists := registry[id]; exists {
		panic(fmt.Sprintf("results: duplicate adapter id %q", id))
	}
	registry[id] = reg
	order = append(order, id)
	for _, a := range aliases {
		if _, exists := registry[a]; exists {
			panic(fmt.Sprintf("results: duplicate adapter alias %q", a))
		}
		registry[a] = reg
	}
}

// Lookup resolves an id or alias to an adapter.
func Lookup(id string) (Adapter, error) {
	reg, ok := registry[id]
	if !ok {
		return nil, fmt.Errorf("unknown test-result format %q (run `testtranslator results formats` to list adapters)", id)
	}
	return reg.adapter, nil
}

// FormatInfo describes a registered adapter for listing.
type FormatInfo struct {
	ID          string   `json:"id"`
	Aliases     []string `json:"aliases,omitempty"`
	Description string   `json:"description"`
}

// Formats returns all registered result adapters in registration order.
func Formats() []FormatInfo {
	out := make([]FormatInfo, 0, len(order))
	for _, id := range order {
		reg := registry[id]
		aliases := append([]string(nil), reg.aliases...)
		sort.Strings(aliases)
		out = append(out, FormatInfo{ID: id, Aliases: aliases, Description: reg.description})
	}
	return out
}
