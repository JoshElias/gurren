package auth

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// ssh -N -L 3307:ground-staging.cluster-clx0ponnomgu.us-west-2.rds.amazonaws.com:3306 ec2-user@bastion-staging

// func getKeyAuth(keyPath string) (ssh.AuthMethod, error) {
// 	key, err := os.ReadFile(keyPath)
// 	if err != nil {
// 		return nil, fmt.Errorf("unable read private key: %v", err)
// 	}
//
// 	// Create the signer
// 	signer, err := ssh.ParsePrivateKey(key)
// 	if err != nil {
// 		return nil, fmt.Errorf("unable to create signer: %v", err)
// 	}
//
// 	return ssh.PublicKeys(signer), nil
// }

func GetAgentAuth() (ssh.AuthMethod, error) {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return nil, errors.New("SSH Agent must be running")
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to ssh-agent: %v", err)
	}

	agentClient := agent.NewClient(conn)

	signers, err := agentClient.Signers()
	if err != nil {
		return nil, fmt.Errorf("unable to get signers: %v", err)
	}

	log.Printf("Agent holding %d keys", len(signers))
	for i, s := range signers {
		log.Printf("  Key %d: %s", i, ssh.FingerprintSHA256(s.PublicKey()))
	}

	return ssh.PublicKeysCallback(agentClient.Signers), nil
}
