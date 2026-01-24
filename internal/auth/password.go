package auth

import (
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// PasswordAuthenticator provides SSH authentication via password.
// TODO: Is storing a password in a config a good idea?
type PasswordAuthenticator struct {
	Password string // Optional: pre-configured password. If empty, prompts user.
}

func (p *PasswordAuthenticator) Name() string {
	return "password"
}

func (p *PasswordAuthenticator) Priority() int {
	return 3 // Lowest priority - last resort
}

func (p *PasswordAuthenticator) IsAvailable() bool {
	// Password auth is always available as a fallback
	return true
}

func (p *PasswordAuthenticator) GetAuthMethod() (ssh.AuthMethod, error) {
	if p.Password != "" {
		return ssh.Password(p.Password), nil
	}

	return ssh.PasswordCallback(func() (string, error) {
		// Prompt for password
		fmt.Print("Enter SSH password: ")
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println() // newline after password input
		if err != nil {
			return "", fmt.Errorf("failed to read password: %w", err)
		}

		return string(password), nil
	}), nil
}
