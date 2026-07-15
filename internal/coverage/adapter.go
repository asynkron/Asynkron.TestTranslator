package coverage

import (
	"fmt"
	"io"
	"sort"

	"github.com/asynkron/Asynkron.TestTranslator/internal/diagnostics"
	"github.com/asynkron/Asynkron.TestTranslator/internal/pathutil"
)

// Options carries conversion configuration common to coverage adapters.
type Options struct {
	// SourceName labels the input in diagnostics.
	SourceName string
	// Paths normalizes source paths into repo-relative POSIX form.
	Paths *pathutil.Normalizer
	// Diag collects warnings and notes.
	Diag *diagnostics.Collector
	// MaxInputBytes bounds bytes read (0 = adapter default).
	MaxInputBytes int64
}

// Adapter parses one exact coverage format into the internal Report model.
type Adapter interface {
	Parse(r io.Reader, opts Options) (*Report, error)
}

type registration struct {
	id          string
	aliases     []string
	description string
	adapter     Adapter
}

var registry = map[string]*registration{}
var order []string

// Register adds a coverage adapter under a canonical id plus optional aliases.
func Register(id, description string, adapter Adapter, aliases ...string) {
	reg := &registration{id: id, aliases: aliases, description: description, adapter: adapter}
	if _, exists := registry[id]; exists {
		panic(fmt.Sprintf("coverage: duplicate adapter id %q", id))
	}
	registry[id] = reg
	order = append(order, id)
	for _, a := range aliases {
		if _, exists := registry[a]; exists {
			panic(fmt.Sprintf("coverage: duplicate adapter alias %q", a))
		}
		registry[a] = reg
	}
}

// Lookup resolves an id or alias to a coverage adapter.
func Lookup(id string) (Adapter, error) {
	reg, ok := registry[id]
	if !ok {
		return nil, fmt.Errorf("unknown coverage format %q (run `testtranslator coverage formats` to list adapters)", id)
	}
	return reg.adapter, nil
}

// FormatInfo describes a registered adapter for listing.
type FormatInfo struct {
	ID          string   `json:"id"`
	Aliases     []string `json:"aliases,omitempty"`
	Description string   `json:"description"`
}

// Formats returns all registered coverage adapters in registration order.
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
