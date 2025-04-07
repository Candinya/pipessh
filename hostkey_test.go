package main

import (
	"bytes"
	"golang.org/x/crypto/ssh"
	"reflect"
	"testing"
)

func Test_extractHostname(t *testing.T) {
	testcases := []struct {
		name                                  string
		hostname                              string
		wantRawHostname, wantFriendlyHostname string
	}{
		{
			name:            "FQDN standard",
			hostname:        "candinya.com:22",
			wantRawHostname: "candinya.com", wantFriendlyHostname: "candinya.com",
		},
		{
			name:            "FQDN non-standard",
			hostname:        "candinya.com:2233",
			wantRawHostname: "[candinya.com]:2233", wantFriendlyHostname: "candinya.com:2233",
		},
		{
			name:            "IPv4 standard",
			hostname:        "192.168.0.1:22",
			wantRawHostname: "192.168.0.1", wantFriendlyHostname: "192.168.0.1",
		},
		{
			name:            "IPv4 non-standard",
			hostname:        "192.168.0.1:2233",
			wantRawHostname: "[192.168.0.1]:2233", wantFriendlyHostname: "192.168.0.1:2233",
		},
		{
			name:            "IPv6 standard",
			hostname:        "[fe80::1]:22",
			wantRawHostname: "fe80::1", wantFriendlyHostname: "fe80::1",
		},
		{
			name:            "IPv6 non-standard",
			hostname:        "[fe80::1]:2233",
			wantRawHostname: "[fe80::1]:2233", wantFriendlyHostname: "[fe80::1]:2233",
		},
	}

	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			hostname, friendlyHostname, err := extractHostname(testcase.hostname)
			if err != nil {
				t.Fatalf("failed to extract hostname: %v", err)
			}

			if hostname != testcase.wantRawHostname {
				t.Errorf("Unexpected hostname: want %q, got %q", testcase.wantRawHostname, hostname)
			}

			if friendlyHostname != testcase.wantFriendlyHostname {
				t.Errorf("Unexpected friendly hostname: want %q, got %q", testcase.wantFriendlyHostname, friendlyHostname)
			}

		})
	}
}

func Test_findServer(t *testing.T) {
	testcases := []struct {
		name                                       string
		knownHosts                                 string
		rawHostname                                string
		rawAddr                                    string
		key                                        string
		wantPerfectMatch                           bool
		wantHostsWithSameKey                       []string
		wantOldKey                                 *string
		wantRelevantLineStart, wantRelevantLineEnd int64
	}{
		{
			name:                  "perfect match single line",
			knownHosts:            "github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl",
			rawHostname:           "github.com",
			rawAddr:               "",
			key:                   "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl",
			wantPerfectMatch:      true,
			wantHostsWithSameKey:  nil,
			wantOldKey:            nil,
			wantRelevantLineStart: 0, wantRelevantLineEnd: 0,
		},
		{
			name:                  "perfect match from batch",
			knownHosts:            "github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl\ngithub.com ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBEmKSENjQEezOmxkZMy7opKgwFB9nkt5YRrYMjNuG5N87uRgg6CLrbo5wAdT/y6v0mKV0U2w0WZ2YB/++Tpockg=\ngithub.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCj7ndNxQowgcQnjshcLrqPEiiphnt+VTTvDP6mHBL9j1aNUkY4Ue1gvwnGLVlOhGeYrnZaMgRK6+PKCUXaDbC7qtbW8gIkhL7aGCsOr/C56SJMy/BCZfxd1nWzAOxSDPgVsmerOBYfNqltV9/hWCqBywINIR+5dIg6JTJ72pcEpEjcYgXkE2YEFXV1JHnsKgbLWNlhScqb2UmyRkQyytRLtL+38TGxkxCflmO+5Z8CSSNY7GidjMIZ7Q4zMjA2n1nGrlTDkzwDCsw+wqFPGQA179cnfGWOWRVruj16z6XyvxvjJwbz0wQZ75XK5tKSb7FNyeIEs4TT4jk+S4dhPeAUC5y+bDYirYgM4GC7uEnztnZyaVWQ7B381AK4Qdrwt51ZqExKbQpTUNn+EjqoTwvqNj4kqx5QUCI0ThS/YkOxJCXmPUWZbhjpCg56i+2aB6CmK2JGhn57K5mj0MNdBXA4/WnwH6XoPWJzK5Nyu2zB3nAZp+S5hpQs+p1vN1/wsjk=",
			rawHostname:           "github.com",
			rawAddr:               "",
			key:                   "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBEmKSENjQEezOmxkZMy7opKgwFB9nkt5YRrYMjNuG5N87uRgg6CLrbo5wAdT/y6v0mKV0U2w0WZ2YB/++Tpockg=",
			wantPerfectMatch:      true,
			wantHostsWithSameKey:  nil,
			wantOldKey:            nil,
			wantRelevantLineStart: 0, wantRelevantLineEnd: 0,
		},
		{
			name:                  "perfect match rawAddr",
			knownHosts:            "192.168.3.117 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl",
			rawHostname:           "",
			rawAddr:               "192.168.3.117",
			key:                   "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl",
			wantPerfectMatch:      true,
			wantHostsWithSameKey:  nil,
			wantOldKey:            nil,
			wantRelevantLineStart: 0, wantRelevantLineEnd: 0,
		},
		{
			name:                  "perfect match non-standrad port",
			knownHosts:            "[192.168.3.117]:2233 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl",
			rawHostname:           "",
			rawAddr:               "[192.168.3.117]:2233",
			key:                   "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl",
			wantPerfectMatch:      true,
			wantHostsWithSameKey:  nil,
			wantOldKey:            nil,
			wantRelevantLineStart: 0, wantRelevantLineEnd: 0,
		},
		{
			name:                  "new server (completely different)",
			knownHosts:            "github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl\n", // with new line
			rawHostname:           "code.nya.work",
			rawAddr:               "",
			key:                   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDE5MJr0aoKPww+tjzyyQpl4G9v2neCw/SUqKg0CmxO7lrOit0oll9dJTXZA4irI0y/04+5SEoZiRbPFcKqgZOAGB6Pl2Z1EPO8xdY/WU5lnkgtzAqCiAwgJ6DvZ3i0+fvYx25lPZmA6rDJTQoTSL5DJd50IXAtf/j415jc46gLirU2ipQKWcGKQpd5LjUAsmPiHfN1AhX93WJqvjyTLQVRp4ufvQxYmJYCxMN2oT6kRbYt2c1J8gxqbQoGjPbdKPnd+2/1bpd8Tovt0mw1/grQ427Cdsel1JypMtz5Tr42kJh3damTn+VjzCqDtEdPhE4Muo4ifFA7zAUgEqosyHyzbUiWsmiKiVeZN+QZyZkd/V9hyE2zr2X7HyW0QD2IuDSh+1Nnp4cycQtC8ejsqi9QWHcRA1+hgExRyPeBQRi1WT1VX7NLbVP3Gk1EdM/MT0GwVARyghvL2nIxANnPNWRE+OEaQp40DvXMIQXbY3oEc5QAeVhKZh7p7aM/wfFtDaB9hJVazWtv5zz/TJXPTGXMmU/HxaZSVbgYI7slpsrlVuaD0Mc+95in9rP9cX3NP4YW3/u4+9/P6F3fSpnWYyYQK6BEBcRAgPpHIofXa4P1YRqVlz20G6QFkntfFlwWppC6g1MZVtB2GqHPt+Td1l+rfE5yNtcy0NXS6Vxfv+cFKQ==",
			wantPerfectMatch:      false,
			wantHostsWithSameKey:  nil,
			wantOldKey:            nil,
			wantRelevantLineStart: 92, wantRelevantLineEnd: 92,
		},
		{
			name:                  "new server (different algo)",
			knownHosts:            "github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl\n", // with new line
			rawHostname:           "github.com",
			rawAddr:               "",
			key:                   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDE5MJr0aoKPww+tjzyyQpl4G9v2neCw/SUqKg0CmxO7lrOit0oll9dJTXZA4irI0y/04+5SEoZiRbPFcKqgZOAGB6Pl2Z1EPO8xdY/WU5lnkgtzAqCiAwgJ6DvZ3i0+fvYx25lPZmA6rDJTQoTSL5DJd50IXAtf/j415jc46gLirU2ipQKWcGKQpd5LjUAsmPiHfN1AhX93WJqvjyTLQVRp4ufvQxYmJYCxMN2oT6kRbYt2c1J8gxqbQoGjPbdKPnd+2/1bpd8Tovt0mw1/grQ427Cdsel1JypMtz5Tr42kJh3damTn+VjzCqDtEdPhE4Muo4ifFA7zAUgEqosyHyzbUiWsmiKiVeZN+QZyZkd/V9hyE2zr2X7HyW0QD2IuDSh+1Nnp4cycQtC8ejsqi9QWHcRA1+hgExRyPeBQRi1WT1VX7NLbVP3Gk1EdM/MT0GwVARyghvL2nIxANnPNWRE+OEaQp40DvXMIQXbY3oEc5QAeVhKZh7p7aM/wfFtDaB9hJVazWtv5zz/TJXPTGXMmU/HxaZSVbgYI7slpsrlVuaD0Mc+95in9rP9cX3NP4YW3/u4+9/P6F3fSpnWYyYQK6BEBcRAgPpHIofXa4P1YRqVlz20G6QFkntfFlwWppC6g1MZVtB2GqHPt+Td1l+rfE5yNtcy0NXS6Vxfv+cFKQ==",
			wantPerfectMatch:      false,
			wantHostsWithSameKey:  nil,
			wantOldKey:            nil,
			wantRelevantLineStart: 92, wantRelevantLineEnd: 92,
		},
		{
			name:                  "same key different host",
			knownHosts:            "github.com,[example.com]:2233,[127.0.0.1]:2233 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl\n", // with new line
			rawHostname:           "code.nya.work",
			rawAddr:               "",
			key:                   "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl",
			wantPerfectMatch:      false,
			wantHostsWithSameKey:  []string{"github.com", "[example.com]:2233", "[127.0.0.1]:2233"},
			wantOldKey:            nil,
			wantRelevantLineStart: 0, wantRelevantLineEnd: 128,
		},
		{
			name:                  "same host new key",
			knownHosts:            "github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl\n", // with new line
			rawHostname:           "github.com",
			rawAddr:               "",
			key:                   "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFKGJvkCkPoissrebkHB17tYjPunEULNKP8fNN6fTQ8M",
			wantPerfectMatch:      false,
			wantHostsWithSameKey:  nil,
			wantOldKey:            p("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl"),
			wantRelevantLineStart: 0, wantRelevantLineEnd: 92,
		},
	}

	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(testcase.key))
			if err != nil {
				t.Fatalf("failed to parse public key: %v", err)
			}

			pm, hwsk, oldk, rls, rle := findServer(bytes.NewReader([]byte(testcase.knownHosts)), testcase.rawHostname, testcase.rawAddr, key)

			if pm != testcase.wantPerfectMatch {
				t.Errorf("Unexpected perfect match: got %t, want %t", pm, testcase.wantPerfectMatch)
			}

			if !reflect.DeepEqual(hwsk, testcase.wantHostsWithSameKey) {
				t.Errorf("Unexpected HostsWithSameKey: expected %q, got %q", testcase.wantHostsWithSameKey, hwsk)
			}

			if testcase.wantOldKey != nil {
				oldkey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(*testcase.wantOldKey))
				if err != nil {
					t.Fatalf("failed to parse public key: %v", err)
				}

				if !bytes.Equal((*oldk).Marshal(), oldkey.Marshal()) {
					t.Errorf("Unexpected OldKey: expected %q, got %q", oldkey.Marshal(), (*oldk).Marshal())
				}
			} else if oldk != nil {
				t.Errorf("Unexpected OldKey: got %q", *oldk)
			}

			if rls != testcase.wantRelevantLineStart {
				t.Errorf("Unexpected RelevantLineStart: expected %d, got %d", testcase.wantRelevantLineStart, rls)
			}

			if rle != testcase.wantRelevantLineEnd {
				t.Errorf("Unexpected RelevantLineEnd: expected %d, got %d", testcase.wantRelevantLineEnd, rle)
			}

		})
	}
}

func Test_spareSpace(t *testing.T) {
	testcases := []struct {
		name string
	}{}

	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

		})
	}
}

func Test_updateKnownHosts(t *testing.T) {
	testcases := []struct {
		name string
	}{}

	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

		})
	}
}
