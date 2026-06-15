package framework

import (
	"os"
	"regexp"
)

var environmentReference = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(:-([^}]*))?\}`)

func expandEnvironment(value string) string {
	return environmentReference.ReplaceAllStringFunc(value, func(reference string) string {
		parts := environmentReference.FindStringSubmatch(reference)
		if environmentValue, ok := os.LookupEnv(parts[1]); ok && environmentValue != "" {
			return environmentValue
		}
		return parts[3]
	})
}
