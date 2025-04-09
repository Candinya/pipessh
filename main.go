package main

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"os"
)

func main() {
	// Prepare basic info
	targetServer, jumpServer, privateKeys, knownHostsFilePath, err := prepare()
	if err != nil {
		LogPanic(fmt.Errorf("failed to prepare: %w", err))
	}

	// Prepare private keys
	var signers []ssh.Signer
	for _, pk := range privateKeys {
		keyBytes, err := os.ReadFile(pk)
		if err != nil {
			LogError(fmt.Errorf("failed to read private key %s: %w", pk, err))
			continue
		}

		signer, err := ssh.ParsePrivateKey(keyBytes)
		if err != nil {
			LogError(fmt.Errorf("failed to parse private key %s: %w", pk, err))
			continue
		}

		signers = append(signers, signer)
	}

	var keyAuth ssh.AuthMethod = nil
	if len(signers) > 0 {
		keyAuth = ssh.PublicKeys(signers...)
	}

	// Configure SSH client
	targetConfig, err := sshConfig(targetServer, keyAuth, knownHostsFilePath)
	if err != nil {
		LogPanic(fmt.Errorf("failed to configure target server: %w", err))
	}

	var jumpConfig *ssh.ClientConfig = nil
	if jumpServer != nil {
		jumpConfig, err = sshConfig(jumpServer, keyAuth, knownHostsFilePath)
		if err != nil {
			LogPanic(fmt.Errorf("failed to configure jump server: %w", err))
		}
	}

	// Dial
	targetClient, jumpClient, err := sshDial(targetServer, targetConfig, jumpServer, jumpConfig)
	if err != nil {
		LogPanic(fmt.Errorf("failed to dial: %w", err))
	}

	if jumpClient != nil {
		defer jumpClient.Close()
	}

	defer targetClient.Close()

	// Create session
	session, err := targetClient.NewSession()
	if err != nil {
		LogPanic(fmt.Errorf("failed to create session: %w", err))
	}
	defer session.Close()

	// Pipe stdin/stdout/stderr
	sshStdIn, err := session.StdinPipe()
	if err != nil {
		LogPanic(fmt.Errorf("failed to pipe stdin: %w", err))
	}
	sshStdout, err := session.StdoutPipe()
	if err != nil {
		LogPanic(fmt.Errorf("failed to pipe stdout: %w", err))
	}
	sshStderr, err := session.StderrPipe()
	if err != nil {
		LogPanic(fmt.Errorf("failed to pipe stderr: %w", err))
	}

	defer sshStdIn.Close()

	// Pipe output to std
	go func() {
		if err := pipe(sshStdout, os.Stdout); err != nil {
			LogPanic(fmt.Errorf("failed to pipe stdout: %w", err))
		}
	}()
	go func() {
		if err := pipe(sshStderr, os.Stderr); err != nil {
			LogPanic(fmt.Errorf("failed to pipe stderr: %w", err))
		}
	}()

	// Pipe input from std
	go func() {
		if err := inPipe(os.Stdin, sshStdIn, session.WindowChange); err != nil {
			LogPanic(fmt.Errorf("failed to in-pipe stdin: %w", err))
		}
	}()

	// Loading finish, start
	startEventBytes, err := buildEvent(EventNameSSHStart, nil)
	if err != nil {
		LogPanic(fmt.Errorf("failed to build start event: %w", err))
	}
	if _, err = os.Stdout.Write(startEventBytes); err != nil {
		LogPanic(fmt.Errorf("failed to write start event: %w", err))
	}

	// Setup terminal
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	// Request pseudo terminal
	if err = session.RequestPty("xterm-256color", 24, 80, modes); err != nil {
		LogPanic(fmt.Errorf("failed to request pty: %w", err))
	}

	// Start remote shell
	if err = session.Shell(); err != nil {
		LogPanic(fmt.Errorf("failed to start shell: %w", err))
	}

	// Wait till end
	if err = session.Wait(); err != nil {
		LogPanic(fmt.Errorf("failed to wait: %w", err))
	}
}
