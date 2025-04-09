package main

import (
	"fmt"
	"golang.org/x/crypto/ssh"
)

func sshConfig(server *Server, keyAuth ssh.AuthMethod, knownHostsFilePath *string) (*ssh.ClientConfig, error) {
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

	cfg := ssh.ClientConfig{
		User:    *server.Username,
		Auth:    authMethods,
		Timeout: DefaultTimeout,
	}

	if knownHostsFilePath != nil {
		cfg.HostKeyCallback = prepareHostKeyHandler(*knownHostsFilePath)
	} else {
		cfg.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	return &cfg, nil
}
