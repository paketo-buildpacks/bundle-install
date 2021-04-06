package bundleinstall

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
)

// GemfileParser parses the Gemfile to determine the version of Ruby used by
// the application.
type GemfileParser struct{}

// NewGemfileParser initializes an instance of GemfileParser.
func NewGemfileParser() GemfileParser {
	return GemfileParser{}
}

// ParseVersion scans the Gemfile for a Ruby version specification.
func (p GemfileParser) ParseVersion(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to parse Gemfile: %w", err)
	}

	quotes := `["']`
	versionOperators := `~>|<|>|<=|>=|=`
	versionNumber := `\d+(\.\d+)?(\.\d+)?`
	expression := fmt.Sprintf(`ruby %s((%s)?\s*%s)%s`, quotes, versionOperators, versionNumber, quotes)
	re := regexp.MustCompile(expression)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		matches := re.FindStringSubmatch(scanner.Text())
		if len(matches) >= 3 {
			return matches[1], nil
		}
	}

	return "", nil
}
