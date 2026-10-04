package results

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const unknownGoTestPackage = "(unknown package)"

const maxGoBuildOutputBytes = 4 << 10

type goTestIdentity struct {
	packageName string
	testName    string
}

type goTestEvent struct {
	Action      string `json:"Action"`
	Package     string `json:"Package"`
	Test        string `json:"Test"`
	ImportPath  string `json:"ImportPath"`
	FailedBuild string `json:"FailedBuild"`
	Output      string `json:"Output"`
}

type lifecycleObservation struct {
	paused       map[goTestIdentity]struct{}
	buildOutput  map[string]string
	failedBuilds map[string]string
	err          error
}

type goTestLifecycle struct {
	latest      string
	hadTerminal bool
}

// parseGoTestJSON observes lifecycle events while the installed translator
// consumes the same stream. The pipe keeps observation bounded: it retains one
// action per test and at most maxGoBuildOutputBytes per build package rather
// than buffering a second copy of the report.
func parseGoTestJSON(input io.Reader) (*Payload, error) {
	reader, writer := io.Pipe()
	observed := make(chan lifecycleObservation, 1)
	go func() {
		observation, err := observeGoTestLifecycle(reader)
		_ = reader.CloseWithError(err)
		observation.err = err
		observed <- observation
	}()

	payload, parseErr := parseTranslatedResults(FormatGoTestJSON, io.TeeReader(input, writer))
	_ = writer.Close()
	observation := <-observed
	if parseErr != nil {
		return nil, parseErr
	}
	if observation.err != nil {
		return nil, fmt.Errorf("observe test lifecycle: %w", observation.err)
	}

	removePausedFailures(payload, observation.paused)
	removePausedObservations(payload, observation.paused)
	preserveBuildFailures(payload, observation)
	// Share enriched compiler output with the saved case as well as gate diagnostics.
	for i := range payload.Observations {
		o := &payload.Observations[i]
		if o.Test != "build" {
			continue
		}
		for _, failure := range payload.Failures {
			if failure.Suite == o.Suite && failure.Test == o.Test {
				o.Message = failure.Message
				break
			}
		}
	}
	return payload, nil
}

func observeGoTestLifecycle(input io.Reader) (lifecycleObservation, error) {
	observation := lifecycleObservation{
		buildOutput:  make(map[string]string),
		failedBuilds: make(map[string]string),
	}
	lifecycles := make(map[goTestIdentity]goTestLifecycle)
	decoder := json.NewDecoder(input)
	for {
		var event goTestEvent
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			return lifecycleObservation{}, err
		}
		switch event.Action {
		case "build-output":
			importPath := goBuildPackage(event.ImportPath, event.Package)
			observation.buildOutput[importPath] = appendBounded(
				observation.buildOutput[importPath], event.Output, maxGoBuildOutputBytes,
			)
		}
		if event.FailedBuild != "" {
			packageName := event.Package
			if packageName == "" {
				packageName = unknownGoTestPackage
			}
			observation.failedBuilds[packageName] = event.FailedBuild
		}
		if event.Test == "" || event.Action == "output" {
			continue
		}
		packageName := event.Package
		if packageName == "" {
			packageName = unknownGoTestPackage
		}
		identity := goTestIdentity{packageName: packageName, testName: event.Test}
		lifecycle := lifecycles[identity]
		lifecycle.latest = event.Action
		lifecycle.hadTerminal = lifecycle.hadTerminal || isTerminalGoTestAction(event.Action)
		lifecycles[identity] = lifecycle
	}

	paused := make(map[goTestIdentity]struct{})
	for identity, lifecycle := range lifecycles {
		if lifecycle.latest == "pause" && !lifecycle.hadTerminal {
			paused[identity] = struct{}{}
		}
	}
	observation.paused = paused
	return observation, nil
}

func goBuildPackage(importPath, packageName string) string {
	if importPath == "" {
		importPath = packageName
	}
	if importPath == "" {
		importPath = unknownGoTestPackage
	}
	return importPath
}

func appendBounded(existing, addition string, limit int) string {
	remaining := limit - len(existing)
	if remaining <= 0 {
		return existing
	}
	if len(addition) <= remaining {
		return existing + addition
	}
	end := 0
	for index := range addition {
		if index > remaining {
			break
		}
		end = index
	}
	return existing + addition[:end]
}

func isTerminalGoTestAction(action string) bool {
	return action == "pass" || action == "fail" || action == "skip"
}

func removePausedFailures(payload *Payload, paused map[goTestIdentity]struct{}) {
	kept := payload.Failures[:0]
	removed := 0
	for _, failure := range payload.Failures {
		identity := goTestIdentity{packageName: failure.Suite, testName: failure.Test}
		if _, ok := paused[identity]; ok {
			removed++
			continue
		}
		kept = append(kept, failure)
	}
	payload.Failures = kept
	payload.Counts.Total -= removed
	payload.Counts.Failed -= removed
}

func preserveBuildFailures(payload *Payload, observation lifecycleObservation) {
	for i := range payload.Failures {
		failure := &payload.Failures[i]
		if failure.Test != "build" {
			continue
		}
		importPath := observation.failedBuilds[failure.Suite]
		if importPath == "" {
			importPath = failure.Suite
		}
		if output := strings.TrimSpace(observation.buildOutput[importPath]); output != "" {
			failure.Message = output
		}
	}
}

func removePausedObservations(payload *Payload, paused map[goTestIdentity]struct{}) {
	kept := payload.Observations[:0]
	for _, observation := range payload.Observations {
		if _, excluded := paused[goTestIdentity{packageName: observation.Suite, testName: observation.Test}]; !excluded {
			kept = append(kept, observation)
		}
	}
	payload.Observations = kept
}
