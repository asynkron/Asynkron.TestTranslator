// Command testtranslator converts test-result and coverage artifacts from many
// runners and coverage tools into JUnit XML and Cobertura XML. main is kept
// minimal: it wires standard IO into the CLI and propagates the exit code.
package main

import (
	"os"

	_ "github.com/asynkron/testtranslator/internal/adapters"
	"github.com/asynkron/testtranslator/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], cli.Streams{
		In:  os.Stdin,
		Out: os.Stdout,
		Err: os.Stderr,
	}))
}
