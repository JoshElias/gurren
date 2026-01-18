package auth

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// Default key paths to check, in order of preference
var defaultKeyPaths = []string{
	"~/.ssh/id_ed25519",
	"~/.ssh/id_ecdsa",
	"~/.ssh/id_rsa",
}

// PublicKeyAuthenticator provides SSH authentication via private key files.
type PublicKeyAuthenticator struct {
	KeyPath string // Optional: specific key path. If empty, checks default locations.
}

func (p *PublicKeyAuthenticator) Name() string {
	return "publickey"
}

func (p *PublicKeyAuthenticator) Priority() int {
	return 2 // Second priority - after agent
}

func (p *PublicKeyAuthenticator) IsAvailable() bool {
	if p.KeyPath != "" {
		path := expandPath(p.KeyPath)
		_, err := os.Stat(path)
		return err == nil
	}

	// Check default locations
	for _, keyPath := range defaultKeyPaths {
		path := expandPath(keyPath)
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

func (p *PublicKeyAuthenticator) GetAuthMethod() (ssh.AuthMethod, error) {
	keyPath := p.KeyPath
	if keyPath == "" {
		// Find first available key
		for _, kp := range defaultKeyPaths {
			path := expandPath(kp)
			if _, err := os.Stat(path); err == nil {
				keyPath = path
				break
			}
		}
	} else {
		keyPath = expandPath(keyPath)
	}

	if keyPath == "" {
		return nil, fmt.Errorf("no private key found")
	}

	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read private key %s: %w", keyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		// Key might be encrypted - try with passphrase
		if _, ok := err.(*ssh.PassphraseMissingError); ok {
			signer, err = p.parseEncryptedKey(key, keyPath)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("unable to parse private key: %w", err)
		}
	}

	return ssh.PublicKeys(signer), nil
}

func (p *PublicKeyAuthenticator) parseEncryptedKey(key []byte, keyPath string) (ssh.Signer, error) {
	fmt.Printf("Enter passphrase for key '%s': ", keyPath)

	passphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // newline after password input

	if err != nil {
		return nil, fmt.Errorf("failed to read passphrase: %w", err)
	}

	signer, err := ssh.ParsePrivateKeyWithPassphrase(key, passphrase)
	if err != nil {
		return nil, fmt.Errorf("unable to parse encrypted private key: %w", err)
	}

	return signer, nil
}

// expandPath expands ~ to the user's home directory
func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}
