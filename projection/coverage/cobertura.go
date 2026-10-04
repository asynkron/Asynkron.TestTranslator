package coverage

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"sort"
	"strings"

	testtranslator "github.com/asynkron/Asynkron.TestTranslator"
	"github.com/asynkron/Asynkron.TestTranslator/internal/xmlguard"
)

// FormatCobertura identifies a native Cobertura XML coverage report.
const FormatCobertura = "cobertura-xml"

// Only the class envelope is decoded here for path classification. Native line,
// branch and method semantics are parsed by the canonical Cobertura adapter.
type coberturaDocument struct {
	XMLName  xml.Name           `xml:"coverage"`
	Sources  []string           `xml:"sources>source"`
	Packages []coberturaPackage `xml:"packages>package"`
}

type coberturaPackage struct {
	Classes []coberturaClass `xml:"classes>class"`
}

type coberturaClass struct {
	Filename string `xml:"filename,attr"`
	Body     string `xml:",innerxml"`
}

type coberturaFile struct {
	path    string
	lines   map[int]testtranslator.LineHit
	methods map[string]bool
}

// ParseCobertura projects Cobertura into coverage.v1. Unsafe paths are disclosed
// while safe classes remain available. Multiple classes for a file are merged
// using the best observed line, branch and method coverage, so a class does not
// inflate the file's structural totals.
func ParseCobertura(reader io.Reader, repoRoot string) (*Report, error) {
	raw, err := readAllBounded(reader, "cobertura xml", maxProjectionInputBytes)
	if err != nil {
		return nil, err
	}
	if err := xmlguard.Check(raw, xmlguard.DefaultMaxDepth); err != nil {
		return nil, fmt.Errorf("cobertura xml: %w", err)
	}
	var document coberturaDocument
	if err := xml.Unmarshal(raw, &document); err != nil {
		return nil, fmt.Errorf("decode Cobertura XML: %w", err)
	}
	report := &Report{
		SchemaVersion: SchemaVersion, RepoRoot: repoRoot,
		SourceTool: "cobertura", SourceFormat: FormatCobertura,
		Files: []FileCoverage{},
	}
	files := map[string]*coberturaFile{}
	for _, pkg := range document.Packages {
		for _, class := range pkg.Classes {
			filename, err := normalizeCoberturaPath(class.Filename, document.Sources, repoRoot)
			if err != nil {
				report.UnavailableReasons = append(report.UnavailableReasons, err.Error())
				continue
			}
			class.Filename = filename
			single := coberturaDocument{Packages: []coberturaPackage{{Classes: []coberturaClass{class}}}}
			encoded, err := xml.Marshal(single)
			if err != nil {
				return nil, fmt.Errorf("encode classified Cobertura class: %w", err)
			}
			native, _, err := testtranslator.ParseCoverage(FormatCobertura, bytes.NewReader(encoded), testtranslator.CoverageOptions{RepoRoot: repoRoot})
			if err != nil {
				return nil, err
			}
			item := files[filename]
			if item == nil {
				item = &coberturaFile{path: filename, lines: map[int]testtranslator.LineHit{}, methods: map[string]bool{}}
				files[filename] = item
			}
			for _, file := range native.Files {
				for _, line := range file.Lines {
					existing := item.lines[line.Number]
					existing.Number = line.Number
					existing.Hits = max(existing.Hits, line.Hits)
					if line.BranchesTotal > existing.BranchesTotal ||
						line.BranchesTotal == existing.BranchesTotal && line.BranchesCovered > existing.BranchesCovered {
						existing.BranchesTotal = line.BranchesTotal
						existing.BranchesCovered = line.BranchesCovered
					}
					item.lines[line.Number] = existing
				}
				for _, function := range file.Functions {
					item.methods[function.Name] = item.methods[function.Name] || function.Hits > 0
				}
			}
		}
	}
	for _, item := range files {
		linesCovered, branchesTotal, branchesCovered := 0, 0, 0
		for _, line := range item.lines {
			if line.Hits > 0 {
				linesCovered++
			}
			branchesTotal += line.BranchesTotal
			branchesCovered += line.BranchesCovered
		}
		functionsTotal, functionsCovered := len(item.methods), 0
		methodNames := make([]string, 0, len(item.methods))
		for name, covered := range item.methods {
			methodNames = append(methodNames, name)
			if covered {
				functionsCovered++
			}
		}
		sort.Strings(methodNames)
		symbols := make([]SymbolCoverage, 0, len(methodNames))
		for _, name := range methodNames {
			symbols = append(symbols, SymbolCoverage{Name: name, Kind: "function", Covered: item.methods[name]})
		}
		file := FileCoverage{
			Path: item.path, Language: coberturaLanguage(item.path),
			LinesTotal: len(item.lines), LinesCovered: linesCovered, LineCoverage: ratio(linesCovered, len(item.lines)),
			FunctionsTotal: &functionsTotal, FunctionsCovered: &functionsCovered, Symbols: symbols,
			SourceTool: report.SourceTool, SourceFormat: report.SourceFormat,
		}
		if branchesTotal > 0 {
			file.BranchesTotal = &branchesTotal
			file.BranchesCovered = &branchesCovered
			file.BranchCoverage = floatPtr(ratio(branchesCovered, branchesTotal))
		}
		report.Files = append(report.Files, file)
	}
	sortFiles(report.Files)
	return report, nil
}

func normalizeCoberturaPath(name string, sources []string, repoRoot string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("cobertura entry has an empty filename")
	}
	name = strings.ReplaceAll(name, `\`, "/")
	candidates := []string{name}
	if !isAbsoluteCoveragePath(name) {
		for _, source := range sources {
			if source = strings.TrimSpace(source); source != "" {
				candidates = append(candidates, path.Join(strings.ReplaceAll(source, `\`, "/"), name))
			}
		}
	}
	for _, candidate := range candidates {
		if relative, err := normalizeKey(candidate, repoRoot, ""); err == nil {
			return relative, nil
		}
	}
	return "", fmt.Errorf("cobertura path %q is outside repository root", name)
}

func coberturaLanguage(file string) string {
	switch strings.ToLower(filepath.Ext(file)) {
	case ".cs":
		return "csharp"
	case ".fs":
		return "fsharp"
	case ".vb":
		return "visual-basic"
	default:
		return ""
	}
}
