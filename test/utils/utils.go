package utils

import (
	"fmt"
	"os"
)

// JUnitFileName Allow adding a prefix into the junit file name.
// prefix using the "TEST_PREFIX" env var
func JUnitFileName(suiteName string) string {
	testPrefix := os.Getenv("TEST_PREFIX")
	if len(testPrefix) > 0 {
		return fmt.Sprintf("junit-%s-%s.xml", testPrefix, suiteName)
	}
	return fmt.Sprintf("junit-%s.xml", suiteName)
}

// SpecDescription Allow adding a prefix into the test spec description.
// prefix using the "TEST_PREFIX" env var
func SpecDescription(spec string) string {
	testPrefix := os.Getenv("TEST_PREFIX")
	if len(testPrefix) > 0 {
		return fmt.Sprintf("%s %s", testPrefix, spec)
	}
	return spec
}
