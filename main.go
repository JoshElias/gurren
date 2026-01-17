package main

import (
	"fmt"
	"io"
	"log"
	"net"

	// "net"

	"golang.org/x/crypto/ssh"
)

func main() {
	fmt.Println("Testing SSH Tunneling")

	bastionAddr := "54.202.126.40:22"
	bastionUser := "ec2-user"
	remoteAddr := "ground-production.cluster-clx0ponnomgu.us-west-2.rds.amazonaws.com:3306"
	localAddr := "127.0.0.1:3307"

	auth, err := GetAgentAuth()
	if err != nil {
		log.Fatal(err)
	}

	config := &ssh.ClientConfig{
		User:            bastionUser,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect to remote ssh server
	bastionClient, err := ssh.Dial("tcp", bastionAddr, config)
	if err != nil {
		log.Fatalf("unable to connect to bastion: %v", err)
	}
	defer func() {
		if err := bastionClient.Close(); err != nil {
			log.Printf("Warning: error closing the bastion client: %v", err)
		}
	}()
	log.Printf("Tunnel Open %s -> %s", localAddr, remoteAddr)

	// Listen on TCP
	localListener, err := net.Listen("tcp", localAddr)
	if err != nil {
		log.Fatalf("unable to listen to bastion connection: %v", err)
	}
	defer func() {
		if err := localListener.Close(); err != nil {
			log.Printf("Warning: error closing the local listener: %v", err)
		}
	}()
	log.Printf("Tunnel Active: localhost:3307 -> %s", remoteAddr)

	for {
		// Listen for connections
		localConn, err := localListener.Accept()
		if err != nil {
			log.Printf("Failed to accept incoming connection: %v", err)
			continue
		}

		go handleForwarding(bastionClient, localConn, remoteAddr)
	}
}

func handleForwarding(sshClient *ssh.Client, localConn net.Conn, remoteAddr string) {
	defer func() {
		if err := localConn.Close(); err != nil {
			log.Printf("unable to close local connection: %v", err)
		}
	}()
	remoteConn, err := sshClient.Dial("tcp", remoteAddr)
	if err != nil {
		log.Printf("failed to dial remote RDS from bastion: %s", err)
		return
	}
	defer func() {
		if err := remoteConn.Close(); err != nil {
			log.Printf("unable to close remote connection: %v", err)
		}
	}()

	// Copy data bi-directionally
	done := make(chan struct{}, 2)

	go func() {
		_, err := io.Copy(remoteConn, localConn)
		if err != nil {
			log.Printf("error sending data over tunnel: %v", err)
		}
		done <- struct{}{}
	}()
	go func() {
		_, err := io.Copy(localConn, remoteConn)
		if err != nil {
			log.Printf("error receiving data from remote: %v", err)
		}
		done <- struct{}{}
	}()

	<-done // Wait for one side to close
}
