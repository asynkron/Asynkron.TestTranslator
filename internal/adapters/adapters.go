// Package adapters blank-imports every bundled adapter so their init functions
// register them with the results and coverage registries. Importing this single
// package wires up the full adapter set without the CLI depending on each
// adapter directly.
package adapters

import (
	// Test-result adapters.
	_ "github.com/asynkron/testtranslator/internal/results/gotest"
	_ "github.com/asynkron/testtranslator/internal/results/nunit"
	_ "github.com/asynkron/testtranslator/internal/results/surefire"
	_ "github.com/asynkron/testtranslator/internal/results/tap"
	_ "github.com/asynkron/testtranslator/internal/results/trx"
	_ "github.com/asynkron/testtranslator/internal/results/xunit"

	// Coverage adapters.
	_ "github.com/asynkron/testtranslator/internal/coverage/coberturain"
	_ "github.com/asynkron/testtranslator/internal/coverage/gocover"
	_ "github.com/asynkron/testtranslator/internal/coverage/istanbul"
	_ "github.com/asynkron/testtranslator/internal/coverage/jacoco"
	_ "github.com/asynkron/testtranslator/internal/coverage/lcov"
)
