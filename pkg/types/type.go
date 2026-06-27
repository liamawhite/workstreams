package types

import (
	"fmt"
	"regexp"
	"strings"
)

var validDirName = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
var consecutiveDashes = regexp.MustCompile(`-{2,}`)

// TypeConfig is the configuration stored in each type's config.yaml.
type TypeConfig struct {
	Name string `yaml:"name"`
}

// ToDirName derives a lowercase-dash directory name from a display name.
func ToDirName(displayName string) string {
	s := strings.ToLower(displayName)
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	s = consecutiveDashes.ReplaceAllString(b.String(), "-")
	return strings.Trim(s, "-")
}

// ValidateDirName returns an error if name is not a valid lowercase-dash type name.
func ValidateDirName(name string) error {
	if !validDirName.MatchString(name) {
		return fmt.Errorf("invalid type name %q: must be lowercase letters, digits, and dashes (e.g. \"my-type\")", name)
	}
	return nil
}
