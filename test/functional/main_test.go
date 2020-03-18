package functional

import (
	"bytes"
	"fmt"
	"github.com/integr8ly/integreatly-operator/test/metadata"
	"github.com/jstemmer/go-junit-report/formatter"
	"github.com/jstemmer/go-junit-report/parser"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testResultsDirectory = "/test-run-results"
	jUnitOutputFilename  = "junit-integreatly-operator.xml"
	addonMetadataName    = "addon-metadata.json"
	testOutputFileName   = "test-output.txt"
)

func captureOutput(f func()) string {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}

	stdout := os.Stdout
	os.Stdout = w
	defer func() {
		os.Stdout = stdout
	}()

	stderr := os.Stderr
	os.Stderr = w
	defer func() {
		os.Stderr = stderr
	}()

	f()
	w.Close()

	var buf bytes.Buffer
	io.Copy(&buf, r)

	return buf.String()
}

func writeOutputToFile(output string, filepath string) error {
	return ioutil.WriteFile(filepath, []byte(output), os.FileMode(0644))
}

func writeJunitReportFile(output string, junitReportPath string) error {
	report, err := parser.Parse(strings.NewReader(output), "")
	if err != nil {
		return err
	}

	file, err := os.Create(junitReportPath)
	if err != nil {
		return err
	}

	defer file.Close()

	err = formatter.JUnitReportXML(report, false, "", file)
	if err != nil {
		return err
	}
	return nil
}

func TestMain(t *testing.M) {
	exitCode := 0

	output := captureOutput(func() {
		exitCode = t.Run()
	})

	fmt.Printf(output)
	//TODO Remove this before merging, used for debugging

	if _, err := os.Stat(testResultsDirectory); !os.IsNotExist(err) {
		err := writeOutputToFile(output, filepath.Join(testResultsDirectory, testOutputFileName))
		if err != nil {
			fmt.Printf("error while writing the test output: %v", err)
			os.Exit(1)
		}

		err = writeJunitReportFile(output, filepath.Join(testResultsDirectory, jUnitOutputFilename))
		if err != nil {
			fmt.Printf("error while writing the junit report file: %v", err)
			os.Exit(1)
		}

		err = metadata.Instance.WriteToJSON(filepath.Join(testResultsDirectory, addonMetadataName))
		if err != nil {
			fmt.Printf("error while writing metadata: %v", err)
			os.Exit(1)
		}
	}

	os.Exit(exitCode)
}
