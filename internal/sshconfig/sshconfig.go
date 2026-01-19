// Package sshconfig provides SSH config file (~/.ssh/config) parsing
// to resolve host aliases to their connection details.
package sshconfig

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/kevinburke/ssh_config"
)

// ResolvedHost contains connection details resolved from SSH config.
type ResolvedHost struct {
	// Hostname is the actual hostname to connect to (from HostName directive, or the alias itself)
	Hostname string
	// User is the username to connect as (from User directive)
	User string
	// Port is the SSH port (from Port directive, defaults to "22")
	Port string
	// IdentityFiles are the private key paths to use (from IdentityFile directives)
	IdentityFiles []string
}

// Resolve looks up a host alias in ~/.ssh/config and /etc/ssh/ssh_config
// and returns the resolved connection details.
//
// If the alias is not found in any SSH config, it returns a ResolvedHost
// with the alias as the Hostname and default values for other fields.
//
// Example SSH config:
//
//	Host bastion-staging
//	    HostName 35.86.41.10
//	    User ec2-user
//	    IdentityFile ~/.ssh/bastion-staging
//
// Calling Resolve("bastion-staging") returns:
//
//	ResolvedHost{
//	    Hostname: "35.86.41.10",
//	    User: "ec2-user",
//	    Port: "22",
//	    IdentityFiles: []string{"~/.ssh/bastion-staging"},
//	}
func Resolve(alias string) *ResolvedHost {
	// Get HostName - if not set, the alias itself is the hostname
	hostname, _ := ssh_config.GetStrict(alias, "HostName")
	if hostname == "" {
		hostname = alias
	}

	// Get User - may be empty if not specified
	user, _ := ssh_config.GetStrict(alias, "User")

	// Get Port - defaults to "22"
	port, _ := ssh_config.GetStrict(alias, "Port")
	if port == "" {
		port = "22"
	}

	// Get IdentityFile(s) - can have multiple
	identityFiles := ssh_config.GetAll(alias, "IdentityFile")

	// Expand ~ in identity file paths
	for i, f := range identityFiles {
		identityFiles[i] = expandPath(f)
	}

	return &ResolvedHost{
		Hostname:      hostname,
		User:          user,
		Port:          port,
		IdentityFiles: identityFiles,
	}
}

// IsFromConfig returns true if the alias was found in SSH config
// (i.e., it has a HostName that differs from the alias, or has other SSH config directives)
func (r *ResolvedHost) IsFromConfig(alias string) bool {
	// If hostname differs from alias, it was definitely resolved from config
	if r.Hostname != alias {
		return true
	}
	// If we have identity files or a user, it came from config
	if len(r.IdentityFiles) > 0 || r.User != "" {
		return true
	}
	// If port differs from default, it came from config
	if r.Port != "22" {
		return true
	}
	return false
}

// Address returns the hostname:port string for connecting
func (r *ResolvedHost) Address() string {
	return r.Hostname + ":" + r.Port
}

// expandPath expands ~ to the user's home directory
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
