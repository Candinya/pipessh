package main

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"net"
	"strconv"
)

func sshDial(targetServer *Server, targetConfig *ssh.ClientConfig, jumpServer *Server, jumpConfig *ssh.ClientConfig) (targetClient *ssh.Client, jumpClient *ssh.Client, err error) {
	targetAddress := net.JoinHostPort(targetServer.Host, strconv.Itoa(targetServer.Port))

	if jumpServer == nil {
		// Connect directly to target server
		targetClient, err = ssh.Dial("tcp", targetAddress, targetConfig)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to connect to target server %s: %w", targetAddress, err)
		}

		return targetClient, nil, nil
	} else {
		// Step 1: Connect to jump server
		jumpAddress := net.JoinHostPort(jumpServer.Host, strconv.Itoa(jumpServer.Port))
		jumpClient, err = ssh.Dial("tcp", jumpAddress, jumpConfig)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to connect to jump server %s: %w", jumpAddress, err)
		}

		// Step 2: Connect to target server
		tnc, err := jumpClient.Dial("tcp", targetAddress)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to dial target server %s: %w", targetAddress, err)
		}

		ncc, chans, reqs, err := ssh.NewClientConn(tnc, targetAddress, targetConfig)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to connect to target server %s: %w", targetAddress, err)
		}

		targetClient = ssh.NewClient(ncc, chans, reqs)

		return targetClient, jumpClient, nil
	}
}
