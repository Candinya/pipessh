package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FlagStringArray []string

// String is an implementation of the flag.Value interface
func (i *FlagStringArray) String() string {
	return fmt.Sprintf("%v", *i)
}

// Set is an implementation of the flag.Value interface
func (i *FlagStringArray) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var (
	flagServerPort int
	flagJumpServer string
	flagIdentity   string
	flagOptions    FlagStringArray
)

func init() {
	flag.IntVar(&flagServerPort, "p", -1, "SSH server port")
	flag.StringVar(&flagJumpServer, "J", "", "Connect through jump server")
	flag.StringVar(&flagIdentity, "i", "", "Authenticate with specific private key")
	flag.Var(&flagOptions, "o", "SSH Options")
}

func prepare() (targetServer *Server, jumpServer *Server, privateKeys []string, knownHostsFilePath *string, err error) {
	// Parse command line args
	flag.Parse()

	commandArgs := flag.Args()
	if len(commandArgs) != 1 {
		// Invalid
		return nil, nil, nil, nil, fmt.Errorf("too many arguments")
	}

	// Parse target server
	targetServer, err = parseServer(commandArgs[0])
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to parse target server: %w", err)
	}
	if targetServer.Username == nil {
		targetServer.Username = p(DefaultUser)
	}
	if flagServerPort != -1 {
		// Valid port overwrite
		targetServer.Port = flagServerPort
	}

	// Parse jump server if any
	if flagJumpServer != "" {
		jumpServer, err = parseServer(flagJumpServer)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("failed to parse jump server: %w", err)
		}
		if jumpServer.Username == nil {
			jumpServer.Username = targetServer.Username
		}
	}

	// Parse identity
	if flagIdentity != "" {
		privateKeys = []string{flagIdentity}
	} else {
		// Find user home to get possible private keys
		homedir, err := os.UserHomeDir()
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("failed to get user home dir: %w", err)
		}
		keyDir := filepath.Join(homedir, ".ssh")
		entries, err := os.ReadDir(keyDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return targetServer, jumpServer, nil, nil, nil
			} else {
				return nil, nil, nil, nil, fmt.Errorf("failed to read SSH keys: %w", err)
			}
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				entryName := entry.Name()
				if strings.HasPrefix(entryName, "id_") && !strings.HasSuffix(entryName, ".pub") {
					// This might be our lucky king
					privateKeys = append(privateKeys, filepath.Join(keyDir, entryName))
				}
			}
		}
	}

	// Parse known_hosts file
	for _, option := range flagOptions {
		optionSep := strings.SplitN(option, "=", 2)
		if len(optionSep) < 2 {
			// Skip invalid option
			continue
		}

		if optionSep[0] == "UserKnownHostsFile" {
			knownHostsFilePath = &optionSep[1]
		}
	}

	if knownHostsFilePath == nil {
		// Default
		homedir, err := os.UserHomeDir()
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("failed to get user home dir: %w", err)
		}
		knownHostsFilePath = p(filepath.Join(homedir, ".ssh", "known_hosts"))
	}

	return targetServer, jumpServer, privateKeys, knownHostsFilePath, nil
}
