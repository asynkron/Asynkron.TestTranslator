package testtranslator_test

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/asynkron/Asynkron.TestTranslator"
)

// Convert a Go `go test -json` stream into JUnit XML in-process.
func ExampleConvertResults() {
	const goTestJSON = `{"Action":"run","Package":"p","Test":"TestA"}
{"Action":"pass","Package":"p","Test":"TestA","Elapsed":0.01}
{"Action":"pass","Package":"p","Elapsed":0.01}
`
	junitXML, diags, err := testtranslator.ConvertResults("go-test-json", strings.NewReader(goTestJSON))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("has testcase:", bytes.Contains(junitXML, []byte(`<testcase name="TestA"`)))
	fmt.Println("diagnostics:", len(diags))
	// Output:
	// has testcase: true
	// diagnostics: 0
}

// Convert an LCOV tracefile into Cobertura XML, rerooting absolute paths.
func ExampleConvertCoverage() {
	const lcov = `SF:/repo/src/a.go
DA:1,3
DA:2,0
LF:2
LH:1
end_of_record
`
	coberturaXML, _, err := testtranslator.ConvertCoverage("lcov", strings.NewReader(lcov), testtranslator.CoverageOptions{
		RepoRoot: "/repo",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("has coverage:", bytes.Contains(coberturaXML, []byte("<coverage")))
	// Output:
	// has coverage: true
}
