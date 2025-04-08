package main

import (
	"bytes"
	"errors"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"
)

func replaceLineSeparator(content string) string {
	return strings.ReplaceAll(content, "\n", LineBreak)
}

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
			wantRelevantLineStart: int64(91 + len(LineBreak)), wantRelevantLineEnd: int64(91 + len(LineBreak)),
		},
		{
			name:                  "new server (completely different) (without newline)",
			knownHosts:            "github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl",
			rawHostname:           "code.nya.work",
			rawAddr:               "",
			key:                   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDE5MJr0aoKPww+tjzyyQpl4G9v2neCw/SUqKg0CmxO7lrOit0oll9dJTXZA4irI0y/04+5SEoZiRbPFcKqgZOAGB6Pl2Z1EPO8xdY/WU5lnkgtzAqCiAwgJ6DvZ3i0+fvYx25lPZmA6rDJTQoTSL5DJd50IXAtf/j415jc46gLirU2ipQKWcGKQpd5LjUAsmPiHfN1AhX93WJqvjyTLQVRp4ufvQxYmJYCxMN2oT6kRbYt2c1J8gxqbQoGjPbdKPnd+2/1bpd8Tovt0mw1/grQ427Cdsel1JypMtz5Tr42kJh3damTn+VjzCqDtEdPhE4Muo4ifFA7zAUgEqosyHyzbUiWsmiKiVeZN+QZyZkd/V9hyE2zr2X7HyW0QD2IuDSh+1Nnp4cycQtC8ejsqi9QWHcRA1+hgExRyPeBQRi1WT1VX7NLbVP3Gk1EdM/MT0GwVARyghvL2nIxANnPNWRE+OEaQp40DvXMIQXbY3oEc5QAeVhKZh7p7aM/wfFtDaB9hJVazWtv5zz/TJXPTGXMmU/HxaZSVbgYI7slpsrlVuaD0Mc+95in9rP9cX3NP4YW3/u4+9/P6F3fSpnWYyYQK6BEBcRAgPpHIofXa4P1YRqVlz20G6QFkntfFlwWppC6g1MZVtB2GqHPt+Td1l+rfE5yNtcy0NXS6Vxfv+cFKQ==",
			wantPerfectMatch:      false,
			wantHostsWithSameKey:  nil,
			wantOldKey:            nil,
			wantRelevantLineStart: int64(91 + len(LineBreak)), wantRelevantLineEnd: int64(91 + len(LineBreak)),
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
			wantRelevantLineStart: int64(91 + len(LineBreak)), wantRelevantLineEnd: int64(91 + len(LineBreak)),
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
			wantRelevantLineStart: 0, wantRelevantLineEnd: int64(127 + len(LineBreak)),
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
			wantRelevantLineStart: 0, wantRelevantLineEnd: int64(91 + len(LineBreak)),
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

			pm, hwsk, oldk, rls, rle := findServer(bytes.NewReader([]byte(replaceLineSeparator(testcase.knownHosts))), testcase.rawHostname, testcase.rawAddr, key)

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
		name                  string
		initialContent        string
		keepBefore, keepAfter int64
		requiredSpace         int64
		wantContent           string
	}{
		{
			name:           "empty",
			initialContent: "",
			keepBefore:     0,
			keepAfter:      0,
			requiredSpace:  0,
			wantContent:    "",
		},
		{
			name:           "longer content 1",
			initialContent: "A123456789B123456789C123456789",
			keepBefore:     5,
			keepAfter:      10,
			requiredSpace:  10,
			wantContent:    "A123456789B1234B123456789C123456789",
		},
		{
			name:           "longer content 2",
			initialContent: "A123456789B123456789C123456789",
			keepBefore:     5,
			keepAfter:      15,
			requiredSpace:  30,
			wantContent:    "A123456789B123456789C123456789\x00\x00\x00\x00\x0056789C123456789",
		},
		{
			name:           "longer content 3",
			initialContent: "A123456789B123456789C123456789",
			keepBefore:     0,
			keepAfter:      15,
			requiredSpace:  30,
			wantContent:    "A123456789B123456789C12345678956789C123456789",
		},
		{
			name:           "shorter content 1",
			initialContent: "A123456789B123456789C123456789",
			keepBefore:     5,
			keepAfter:      10,
			requiredSpace:  2,
			wantContent:    "A123456B123456789C123456789",
		},
		{
			name:           "shorter content 2",
			initialContent: "A123456789B123456789C123456789",
			keepBefore:     5,
			keepAfter:      30,
			requiredSpace:  2,
			wantContent:    "A123456",
		},
		{
			name:           "shorter content 3",
			initialContent: "A123456789B123456789C123456789",
			keepBefore:     5,
			keepAfter:      30,
			requiredSpace:  0,
			wantContent:    "A1234",
		},
		{
			name:           "same length",
			initialContent: "A123456789B123456789C123456789",
			keepBefore:     5,
			keepAfter:      15,
			requiredSpace:  10,
			wantContent:    "A123456789B123456789C123456789",
		},
	}

	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			// Prepare
			f, err := os.CreateTemp("", "pipessh-test-sparespace.*.txt")
			if err != nil {
				log.Fatal(err)
			}
			defer os.Remove(f.Name()) // clean up

			if _, err = f.WriteString(testcase.initialContent); err != nil {
				t.Fatalf("failed to write content: %v", err)
			}

			// Test
			if err = spareSpace(f, testcase.keepBefore, testcase.keepAfter, testcase.requiredSpace); err != nil {
				t.Fatalf("failed to spareSpace: %v", err)
			}

			// Validate
			buf := make([]byte, DefaultBufferSize)

			readBytes, err := f.ReadAt(buf, 0)
			if err != nil && !errors.Is(err, io.EOF) {
				t.Fatalf("failed to read content: %v", err)
			}

			if readBytes != len(testcase.wantContent) {
				t.Errorf("Unexpected file size: expected %d, got %d", len(testcase.wantContent), readBytes)
			}

			if !bytes.Equal(buf[:readBytes], []byte(testcase.wantContent)) {
				t.Errorf("Unexpected content: expected %q, got %q", testcase.wantContent, string(buf[:readBytes]))
			}
		})
	}
}

func Test_updateKnownHosts(t *testing.T) {
	testcases := []struct {
		name                               string
		initialContent                     string
		rawHostname                        string
		key                                string
		oldKey                             *string
		hostsWithSameKey                   []string
		relevantLineStart, relevantLineEnd int64
		wantContent                        string
	}{
		{
			name:              "new host empty file",
			initialContent:    "",
			rawHostname:       "github.com",
			key:               "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFKGJvkCkPoissrebkHB17tYjPunEULNKP8fNN6fTQ8M",
			oldKey:            nil,
			hostsWithSameKey:  nil,
			relevantLineStart: 0, relevantLineEnd: 0,
			wantContent: "github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFKGJvkCkPoissrebkHB17tYjPunEULNKP8fNN6fTQ8M\n",
		},
		{
			name:              "new host non-empty file",
			initialContent:    "github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl\n",
			rawHostname:       "candinya.com",
			key:               "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEvllz8Y+okFQ2M/64Wf4PSxJbV31WHA7/CXFPNhTaMd",
			oldKey:            nil,
			hostsWithSameKey:  nil,
			relevantLineStart: int64(91 + len(LineBreak)), relevantLineEnd: int64(91 + len(LineBreak)),
			wantContent: "github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl\ncandinya.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEvllz8Y+okFQ2M/64Wf4PSxJbV31WHA7/CXFPNhTaMd\n",
		},
		{
			name:              "new host non-empty file (without newline)",
			initialContent:    "github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl",
			rawHostname:       "candinya.com",
			key:               "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEvllz8Y+okFQ2M/64Wf4PSxJbV31WHA7/CXFPNhTaMd",
			oldKey:            nil,
			hostsWithSameKey:  nil,
			relevantLineStart: int64(91 + len(LineBreak)), relevantLineEnd: int64(91 + len(LineBreak)),
			wantContent: "github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl\ncandinya.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEvllz8Y+okFQ2M/64Wf4PSxJbV31WHA7/CXFPNhTaMd\n",
		},
		{
			name:              "new host same key",
			initialContent:    "github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl\n",
			rawHostname:       "candinya.com",
			key:               "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl",
			oldKey:            nil,
			hostsWithSameKey:  []string{"github.com"},
			relevantLineStart: 0, relevantLineEnd: int64(91 + len(LineBreak)),
			wantContent: "github.com,candinya.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl\n",
		},
		{
			name:              "new host (custom port) same key (without newline)",
			initialContent:    "github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl",
			rawHostname:       "[candinya.com]:2233",
			key:               "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl",
			oldKey:            nil,
			hostsWithSameKey:  []string{"github.com"},
			relevantLineStart: 0, relevantLineEnd: int64(91 + len(LineBreak)),
			wantContent: "github.com,[candinya.com]:2233 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl\n",
		},
		{
			name:              "new host (multiple) same key",
			initialContent:    "github.com,[example.com]:2233,[127.0.0.1]:2233 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl\n",
			rawHostname:       "candinya.com",
			key:               "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl",
			oldKey:            nil,
			hostsWithSameKey:  []string{"github.com", "[example.com]:2233", "[127.0.0.1]:2233"},
			relevantLineStart: 0, relevantLineEnd: int64(127 + len(LineBreak)),
			wantContent: "github.com,[example.com]:2233,[127.0.0.1]:2233,candinya.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl\n",
		},
		{
			name:              "old host new key",
			initialContent:    "candinya.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl\n",
			rawHostname:       "candinya.com",
			key:               "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFKGJvkCkPoissrebkHB17tYjPunEULNKP8fNN6fTQ8M",
			oldKey:            p("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl"),
			hostsWithSameKey:  nil,
			relevantLineStart: 0, relevantLineEnd: int64(93 + len(LineBreak)),
			wantContent: "candinya.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFKGJvkCkPoissrebkHB17tYjPunEULNKP8fNN6fTQ8M\n",
		},
	}

	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			// Prepare
			f, err := os.CreateTemp("", "pipessh-test-updateKnownHosts.*.txt")
			if err != nil {
				log.Fatal(err)
			}
			defer os.Remove(f.Name()) // clean up

			if _, err = f.WriteString(replaceLineSeparator(testcase.initialContent)); err != nil {
				t.Fatalf("failed to write content: %v", err)
			}

			key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(testcase.key))
			if err != nil {
				t.Fatalf("failed to parse public key: %v", err)
			}

			var oldKey *ssh.PublicKey = nil
			if testcase.oldKey != nil {
				oldKeyData, _, _, _, err := ssh.ParseAuthorizedKey([]byte(*testcase.oldKey))
				if err != nil {
					t.Fatalf("failed to parse public key: %v", err)
				}
				oldKey = &oldKeyData
			}

			// Test
			if err = updateKnownHosts(f, testcase.rawHostname, key, oldKey, testcase.hostsWithSameKey, testcase.relevantLineStart, testcase.relevantLineEnd); err != nil {
				t.Fatalf("failed to updateKnownHosts: %v", err)
			}

			// Validate
			buf := make([]byte, DefaultBufferSize)

			readBytes, err := f.ReadAt(buf, 0)
			if err != nil && !errors.Is(err, io.EOF) {
				t.Fatalf("failed to read content: %v", err)
			}

			wantContentWithPlatformSpecifiedLineSeparator := replaceLineSeparator(testcase.wantContent)

			if readBytes != len(wantContentWithPlatformSpecifiedLineSeparator) {
				t.Errorf("Unexpected file size: expected %d, got %d", len(wantContentWithPlatformSpecifiedLineSeparator), readBytes)
			}

			if !bytes.Equal(buf[:readBytes], []byte(wantContentWithPlatformSpecifiedLineSeparator)) {
				t.Errorf("Unexpected content: expected %q, got %q", wantContentWithPlatformSpecifiedLineSeparator, string(buf[:readBytes]))
			}

		})
	}
}
