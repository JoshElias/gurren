package auth

import "testing"

func TestGetAuthenticators(t *testing.T) {
	authenticators := GetAllAuthenticators()
	if len(authenticators) == 0 {
		t.Error("No authenticators defined")
	}
}

func TestUniqueAuthenticatorPriority(t *testing.T) {
	seen := make(map[int]string)

	for _, auth := range GetAllAuthenticators() {
		if existing, exists := seen[auth.Priority()]; exists {
			t.Errorf("priority %d used by both %s and %s", auth.Priority(), existing, auth.Name())
		}
	}
}
