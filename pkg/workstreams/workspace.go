package workstreams

import (
	"fmt"
	"regexp"
	"strings"
)

var validDirName = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

var consecutiveDashes = regexp.MustCompile(`-{2,}`)

// Config is the structure written to config.yaml inside each workstream directory.
type Config struct {
	Name     string            `yaml:"name"`
	Template string            `yaml:"template,omitempty"`
	Links    map[string]string `yaml:"links"`
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
	s = strings.Trim(s, "-")
	return s
}

// ValidateDirName returns an error if name is not a valid lowercase-dash workstream name.
func ValidateDirName(name string) error {
	if !validDirName.MatchString(name) {
		return fmt.Errorf("invalid workstream directory name %q: must be lowercase letters, digits, and dashes (e.g. \"my-project\")", name)
	}
	return nil
}
