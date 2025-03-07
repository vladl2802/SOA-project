package handles

import (
	"errors"
	"strings"
	"testing"
)

func TestCheckPasswordValidity(t *testing.T) {
	tests := []struct {
		name     string
		password string
		expected error
	}{
		{
			name:     "Valid",
			password: "ValidPass1!",
			expected: nil,
		},
		{
			name:     "Too short",
			password: "short",
			expected: errors.New("too short password"),
		},
		{
			name:     "Too long",
			password: strings.Repeat("A", 73),
			expected: errors.New("too long password"),
		},
		{
			name:     "Non-ASCII",
			password: "Pasлолrd_1",
			expected: errors.New("non-ASCII character in password"),
		},
		{
			name:     "Spaces",
			password: "Space space",
			expected: errors.New("space in password"),
		},
		{
			name:     "No digits",
			password: "NoDigits!!",
			expected: errors.New("password doesn't contain enough variety of symbols"),
		},
		{
			name:     "No letters",
			password: "1234567!!",
			expected: errors.New("password doesn't contain enough variety of symbols"),
		},
		{
			name:     "No special",
			password: "NoSpecial1",
			expected: errors.New("password doesn't contain enough variety of symbols"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkPasswordValidity(tt.password)
			if (err == nil && tt.expected != nil) || (err != nil && tt.expected == nil) || (err != nil && err.Error() != tt.expected.Error()) {
				t.Errorf("checkPasswordValidity(%q) returned %v, where %v expected", tt.password, err, tt.expected)
			}
		})
	}
}
