package bundle

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
)

type GemfileParser struct{}

func NewGemfileParser() GemfileParser {
	return GemfileParser{}
}

func (p GemfileParser) ParseVersion(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}

		return "", fmt.Errorf("failed to parse Gemfile: %w", err)
	}

	quotes := `["']`
	versionOperators := `~>|<|>|<=|>=|=`
	versionNumber := `\d+\.\d+\.\d+`
	expression := fmt.Sprintf(`ruby %s((%s)?\s*%s)%s`, quotes, versionOperators, versionNumber, quotes)
	re := regexp.MustCompile(expression)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		matches := re.FindStringSubmatch(scanner.Text())

		if len(matches) == 3 {
			return matches[1], nil
		}
	}

	return "", nil
}
