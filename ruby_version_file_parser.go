package bundleinstall

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// RubyVersionFileParser parses the .ruby-version file to determine the
// version of Ruby used by the application.
type RubyVersionFileParser struct{}

// NewGemfileParser initializes an instance of RubyVersionFileParser.
func NewRubyVersionFileParser() RubyVersionFileParser {
	return RubyVersionFileParser{}
}

// ParseVersion scans the .ruby-version file for a Ruby version specification.
func (p RubyVersionFileParser) ParseVersion(path string) (string, error) {
	rVersion, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read .ruby-version file: %w", err)
	}

	re := regexp.MustCompile(versionNumberExpression)
	rubyVersion := re.FindString(strings.TrimSpace(string(rVersion)))

	if len(rubyVersion) == 0 {
		return "", fmt.Errorf("no valid ruby version found in .ruby-version file: %s", rVersion)
	}

	return rubyVersion, nil
}
