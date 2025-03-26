package main

import (
	"errors"
	"strings"
	"testing"
)

func TestCheckLoginCorrectness(t *testing.T) {
	tests := []struct {
		name     string
		login    string
		expected error
	}{
		{
			name:     "Valid",
			login:    "V4l1d-l0g1n",
			expected: nil,
		},
		{
			name:     "Empty",
			login:    "",
			expected: errors.New("empty login"),
		},
		{
			name:     "With spaces",
			login:    "invalid login",
			expected: errors.New("login must not contain spaces"),
		},
		{
			name:     "Too long",
			login:    strings.Repeat("a", 101),
			expected: errors.New("too long login"),
		},
		{
			name:     "Invalid UTF-8",
			login:    string([]byte{0xff, 0xfe, 0xfd}),
			expected: errors.New("not valid utf8"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkLoginCorrectness(tt.login)
			if (err == nil && tt.expected != nil) || (err != nil && tt.expected == nil) || (err != nil && err.Error() != tt.expected.Error()) {
				t.Errorf("checkLoginCorrectness(%q) returned %v, where %v expected", tt.login, err, tt.expected)
			}
		})
	}
}
