package workspace

import "testing"

func TestToDirName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"My Project", "my-project"},
		{"foo bar  baz", "foo-bar-baz"},
		{"Already-Lowercase", "already-lowercase"},
		{"Hello World!", "hello-world"},
		{"  leading trailing  ", "leading-trailing"},
		{"café au lait", "caf-au-lait"},
		{"a", "a"},
		{"123 Numbers", "123-numbers"},
		{"!!!", ""},
		{"My  Multiple   Spaces", "my-multiple-spaces"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ToDirName(tt.input)
			if got != tt.want {
				t.Errorf("ToDirName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateDirName(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"my-project", false},
		{"foo", false},
		{"a1b2", false},
		{"foo-bar-baz", false},
		{"a-b-c", false},
		{"", true},
		{"My-Project", true},
		{"foo_bar", true},
		{"-foo", true},
		{"foo-", true},
		{"foo--bar", true},
		{"123", true},
		{"foo bar", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := ValidateDirName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDirName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
