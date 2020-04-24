package bundler

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type GemfileLockParser struct{}

func NewGemfileLockParser() GemfileLockParser {
	return GemfileLockParser{}
}

func (p GemfileLockParser) ParseVersion(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}

		return "", fmt.Errorf("failed to parse Gemfile.lock: %w", err)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "BUNDLED WITH" {
			if scanner.Scan() {
				return strings.TrimSpace(scanner.Text()), nil
			}
		}
	}

	return "", nil
}
