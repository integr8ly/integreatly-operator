package resources

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

// Cut string size, but maintain a reference to the original string using a hash of the full string in the result
func ShortenString(s string, n int) string {
	hashLen := 4

	if len(s) < n {
		return s
	}
	// +1 to account for the hyphen
	postfixLen := hashLen + 1
	cutSize := n - postfixLen

	if n < (hashLen + 1) {
		n = len(s)
		cutSize = len(s)
	}
	cutStr := s[0:cutSize]

	hasher := sha256.New()
	if _, err := hasher.Write([]byte(s)); err != nil {
		return ""
	}
	hashedStr := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

	return strings.ToLower(fmt.Sprintf("%s-%s", cutStr, hashedStr[0:hashLen]))
}
