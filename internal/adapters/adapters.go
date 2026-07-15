// Package adapters blank-imports every bundled adapter so their init functions
// register them with the results and coverage registries. Importing this single
// package wires up the full adapter set without the CLI depending on each
// adapter directly.
package adapters

import (
	// Test-result adapters.
	_ "github.com/asynkron/Asynkron.TestTranslator/internal/results/gotest"
	_ "github.com/asynkron/Asynkron.TestTranslator/internal/results/nunit"
	_ "github.com/asynkron/Asynkron.TestTranslator/internal/results/surefire"
	_ "github.com/asynkron/Asynkron.TestTranslator/internal/results/tap"
	_ "github.com/asynkron/Asynkron.TestTranslator/internal/results/trx"
	_ "github.com/asynkron/Asynkron.TestTranslator/internal/results/xunit"

	// Coverage adapters.
	_ "github.com/asynkron/Asynkron.TestTranslator/internal/coverage/coberturain"
	_ "github.com/asynkron/Asynkron.TestTranslator/internal/coverage/gocover"
	_ "github.com/asynkron/Asynkron.TestTranslator/internal/coverage/istanbul"
	_ "github.com/asynkron/Asynkron.TestTranslator/internal/coverage/jacoco"
	_ "github.com/asynkron/Asynkron.TestTranslator/internal/coverage/lcov"
)
