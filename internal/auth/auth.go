// Package auth manages the available authentication methods
package auth

import (
	"fmt"
	"sort"

	"golang.org/x/crypto/ssh"
)

// Authenticator defines the interface for SSH authentication methods.
type Authenticator interface {
	Name() string
	GetAuthMethod() (ssh.AuthMethod, error)
	IsAvailable() bool
	Priority() int // Lower = higher priority (tried first)
}

func GetAllAuthenticators() []Authenticator {
	authenticators := []Authenticator{
		&AgentAuthenticator{},
		&PublicKeyAuthenticator{},
		&PasswordAuthenticator{},
	}
	return authenticators
}

func GetAvailableAuthenticators() []Authenticator {
	all := GetAllAuthenticators()
	var available []Authenticator
	for _, auth := range all {
		if auth.IsAvailable() {
			available = append(available, auth)
		}
	}
	return available
}

func GetAvailableAuthMethods() ([]ssh.AuthMethod, error) {
	authenticators := GetAvailableAuthenticators()
	if len(authenticators) == 0 {
		return nil, fmt.Errorf("no authentication methods available")
	}

	// Sort by priority (lower = first)
	sort.Slice(authenticators, func(a, b int) bool {
		return authenticators[a].Priority() < authenticators[b].Priority()
	})

	// Collect auth methods
	var methods []ssh.AuthMethod
	for _, auth := range authenticators {
		m, err := auth.GetAuthMethod()
		if err != nil {
			// Silent failure in auto mode - skip to next
			continue
		}
		methods = append(methods, m)
	}

	if len(methods) == 0 {
		return nil, fmt.Errorf("no authentication methods could be initialized")
	}

	return methods, nil
}

// GetAuthMethodsByName returns SSH auth methods based on the specified method.
// If method is "auto", returns all available methods sorted by priority.
// Otherwise, returns only the specified method.
func GetAuthMethodsByName(method string) ([]ssh.AuthMethod, error) {
	methods, err := GetAvailableAuthMethods()
	if err != nil {
		return nil, err
	}

	if method == "" || method == "auto" {
		return methods, nil
	}

	return getAuthMethodByName(method)
}

func getAuthMethodByName(method string) ([]ssh.AuthMethod, error) {
	authenticators := GetAllAuthenticators()
	for _, auth := range authenticators {
		if auth.Name() == method {
			if !auth.IsAvailable() {
				return nil, fmt.Errorf("authentication method %q is not available", method)
			}
			m, err := auth.GetAuthMethod()
			if err != nil {
				return nil, fmt.Errorf("failed to initialize %q auth: %w", method, err)
			}
			return []ssh.AuthMethod{m}, nil
		}
	}
	return nil, fmt.Errorf("unknown authentication method: %q", method)
}
