package main

import (
	"fmt"
	"strconv"
	"strings"
)

func parseServer(connStr string) (*Server, error) {
	s := new(Server)
	var err error

	userAndServerSplits := strings.SplitN(connStr, "@", 2)

	// The former: user info (optional)
	if len(userAndServerSplits) > 1 {
		userInfo := strings.SplitN(userAndServerSplits[0], ":", 2)
		s.Username = &userInfo[0]
		if len(userInfo) > 1 {
			s.Password = &userInfo[1]
		}
	}

	// The latter: server info
	serverStr := userAndServerSplits[len(userAndServerSplits)-1]

	if strings.Count(serverStr, ":") > 1 {
		// Includes IPv6
		if serverStr[0] == '[' { // Special check for IPv6 looks like [fe80::1]:22
			serverInfo := strings.SplitN(serverStr, "]", 2)
			s.Host = serverInfo[0][1:] // Remove leading square bracket
			if len(serverInfo) > 1 && serverInfo[1] != "" {
				if s.Port, err = strconv.Atoi(serverInfo[1][1:]); err != nil { // Remove colon
					return nil, fmt.Errorf("invalid server port %s", serverInfo[1])
				}
			} else {
				// Only IPv6, no port
				s.Port = DefaultSSHPort
			}
		} else {
			// Just raw IPv6
			s.Host = serverStr
			s.Port = DefaultSSHPort
		}
	} else {
		// FQDN or IPv4
		serverInfo := strings.SplitN(serverStr, ":", 2)
		s.Host = serverInfo[0]
		if len(serverInfo) > 1 {
			if s.Port, err = strconv.Atoi(serverInfo[1]); err != nil {
				return nil, fmt.Errorf("invalid server port %s", serverInfo[1])
			}
		} else {
			s.Port = DefaultSSHPort
		}
	}

	return s, nil
}
