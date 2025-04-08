package main

import (
	"fmt"
	"golang.org/x/crypto/ssh"
)

func sshConfig(server *Server, keyAuth ssh.AuthMethod) (*ssh.ClientConfig, error) {
	var authMethods []ssh.AuthMethod
	if server.Password != nil {
		authMethods = append(authMethods, ssh.Password(*server.Password))
	}
	if keyAuth != nil {
		authMethods = append(authMethods, keyAuth)
	}
	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no auth methods found")
	}

	return &ssh.ClientConfig{
		User: *server.Username,
		Auth: authMethods,
		//HostKeyCallback: hostKeyHandler,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         DefaultTimeout,
	}, nil
}
