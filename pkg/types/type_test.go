package types

import "testing"

func TestToDirName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"My Type", "my-type"},
		{"golang", "golang"},
		{"Go Lang", "go-lang"},
		{"  spaced  ", "spaced"},
		{"café", "caf"},
		{"foo--bar", "foo-bar"},
		{"foo_bar", "foobar"},
		{"123 start", "123-start"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ToDirName(tt.input); got != tt.want {
				t.Errorf("ToDirName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateDirName(t *testing.T) {
	valid := []string{"golang", "my-type", "go-lang-2", "a", "type1"}
	for _, name := range valid {
		if err := ValidateDirName(name); err != nil {
			t.Errorf("ValidateDirName(%q) unexpected error: %v", name, err)
		}
	}

	invalid := []string{"", "My-Type", "my_type", "-foo", "foo-", "foo--bar", "1start"}
	for _, name := range invalid {
		if err := ValidateDirName(name); err == nil {
			t.Errorf("ValidateDirName(%q) expected error, got nil", name)
		}
	}
}
