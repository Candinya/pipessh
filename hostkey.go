package main

import (
	"bufio"
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
)

func hostKeyHandler(hostname string, remote net.Addr, key ssh.PublicKey) error {
	rawHostname, friendlyHostname, err := extractHostname(hostname)
	if err != nil {
		return fmt.Errorf("failed to extract hostname %s: %w", hostname, err)
	}

	var rawAddr string
	if remote.(*net.TCPAddr).Port == DefaultSSHPort {
		rawAddr = remote.(*net.TCPAddr).IP.String()
	} else {
		rawAddr = fmt.Sprintf("[%s]:%d", remote.(*net.TCPAddr).IP.String(), remote.(*net.TCPAddr).Port)
	}

	// Query known_hosts file
	homedir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home dir: %w", err)
	}
	knownHostsFilePath := filepath.Join(homedir, ".ssh", "known_hosts")

	knownHostsFile, err := os.OpenFile(knownHostsFilePath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("failed to open known_hosts file: %w", err)
	}

	defer knownHostsFile.Close()

	isPerfectMatch, hostsWithSameKey, oldKey, relevantLineStart, relevantLineEnd := findServer(knownHostsFile, rawHostname, rawAddr, key)
	if isPerfectMatch {
		return nil
	}

	// No matching result found, prepare event
	evPayload := EventPayloadHostKey{
		Host:      friendlyHostname,
		PublicKey: string(key.Marshal()),
	}

	if oldKey == nil {
		// New host
		evPayload.HostWithSameKey = hostsWithSameKey // Could be nil, but that's expected
	} else {
		// Server change its key
		evPayload.OldPublicKey = p(string((*oldKey).Marshal()))
	}

	// Send event
	keyEvBytes, err := buildEvent(EventNameHostKey, &evPayload)
	if err != nil {
		return fmt.Errorf("failed to build key event: %w", err)
	}
	if _, err = os.Stdout.Write(keyEvBytes); err != nil {
		return fmt.Errorf("failed to write key event: %w", err)
	}

	// Waiting for reply
	resBuf := make([]byte, DefaultBufferSize)
	n, err := os.Stdin.Read(resBuf)
	if err != nil {
		return fmt.Errorf("failed to read from stdin: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("nothing read from stdin")
	}
	if !arrayContains([]byte("yY1\r\n"), resBuf[0]) {
		// User rejected
		return fmt.Errorf("user rejected")
	}

	// else: user approved, update file before proceed
	if err = updateKnownHosts(knownHostsFile, rawHostname, key, oldKey, hostsWithSameKey, relevantLineStart, relevantLineEnd); err != nil {
		// Update failed, but continue processing
		LogError(fmt.Errorf("failed to update known_hosts file: %w", err))
	}

	return nil
}

func extractHostname(hostname string) (rawHostname, friendlyHostname string, err error) {
	// hostname will always include port
	host, port, err := net.SplitHostPort(hostname)
	if err != nil {
		return "", "", fmt.Errorf("failed to split host: %v", err)
	}
	if port == fmt.Sprintf("%d", DefaultSSHPort) {
		rawHostname = host
		friendlyHostname = host // discard port
	} else {
		rawHostname = fmt.Sprintf("[%s]:%s", host, port) // Always include square bracket when using non-standard port
		friendlyHostname = hostname
	}
	return
}

func findServer(knownHostsFile io.Reader, hostname string, rawAddr string, key ssh.PublicKey) (bool, []string, *ssh.PublicKey, int64, int64) {
	var (
		relevantLineStart int64 = 0
		relevantLineEnd   int64 = 0
	)

	knownHostsScanner := bufio.NewScanner(knownHostsFile)
	for ; knownHostsScanner.Scan(); relevantLineStart = relevantLineEnd {
		line := knownHostsScanner.Text()

		if len(strings.TrimSpace(line)) == 0 {
			// Empty line
			continue
		}

		relevantLineEnd += int64(len(line)) + 1 // +1 for separator

		// Each line: host1:port1,host2,host3... algo pubkey
		// for example:
		// github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl
		splits := strings.SplitN(line, " ", 2)
		if len(splits) != 2 {
			// Malformed line, skip
			continue
		}

		// Parse
		hostsInLine := strings.Split(splits[0], ",")
		keyInLine, _, _, _, err := ssh.ParseAuthorizedKey([]byte(splits[1]))
		if err != nil {
			// Parse failed, skip
			continue
		}

		// Compare
		isHostMatch := (arrayContains(hostsInLine, hostname) || arrayContains(hostsInLine, rawAddr)) && (key.Type() == keyInLine.Type())
		isKeyMatch := bytes.Equal(key.Marshal(), keyInLine.Marshal())

		if !isHostMatch && !isKeyMatch {
			// Not this one, proceed next line
			continue
		} else if isHostMatch && isKeyMatch {
			// Perfect Match
			return true, nil, nil, 0, 0
		} else if isHostMatch { // !isKeyMatch
			// Server change its key
			return false, nil, &keyInLine, relevantLineStart, relevantLineEnd
		} else { // isKeyMatch && !isHostMatch
			// Access the same server using different host
			return false, hostsInLine, nil, relevantLineStart, relevantLineEnd
		}

	}

	// Nothing matches, this is a new server
	return false, nil, nil, relevantLineStart, relevantLineEnd // Use relevant line end to mark file end position
}

func updateKnownHosts(knownHostsFile *os.File, hostname string, key ssh.PublicKey, oldKey *ssh.PublicKey, hostsWithSameKey []string, relevantLineStart, relevantLineEnd int64) error {
	bytesToWrite := []byte(fmt.Sprintf("%s %s\n", strings.Join(append(hostsWithSameKey, hostname), ","), key.Marshal()))
	if oldKey == nil && hostsWithSameKey == nil {
		// Brand-new host, just append to end of file
		if _, err := knownHostsFile.Seek(-1, io.SeekEnd); err != nil {
			return fmt.Errorf("failed to seek known_hosts file: %w", err)
		}
		finalByte := make([]byte, 1)
		if _, err := knownHostsFile.Read(finalByte); err != nil {
			return fmt.Errorf("failed to read final byte of known_hosts file: %w", err)
		}
		if finalByte[0] != '\n' {
			// No line separator at the end of file, should add line separator before our content or file would be corrupted
			bytesToWrite = append([]byte{'\n'}, bytesToWrite...)
		}
		if _, err := knownHostsFile.Write(bytesToWrite); err != nil {
			return fmt.Errorf("failed to append to known_hosts file: %w", err)
		}
	} else {
		// Partial modification
		// Step 1: Spare space
		if _, err := knownHostsFile.Seek(relevantLineStart, io.SeekStart); err != nil {
			return fmt.Errorf("failed to seek known_hosts file: %w", err)
		}

		// Get file current size
		var fileSize int64
		if stat, err := knownHostsFile.Stat(); err != nil {
			return fmt.Errorf("failed to stat known_hosts file: %w", err)
		} else {
			fileSize = stat.Size()
		}

		// Calculate length difference
		if err := spareSpace(knownHostsFile, fileSize, relevantLineStart, relevantLineEnd, int64(len(bytesToWrite))); err != nil {
			return fmt.Errorf("failed to space space from known_hosts file: %w", err)
		}

		// Step 2: write
		if _, err := knownHostsFile.WriteAt(bytesToWrite, relevantLineStart); err != nil {
			return fmt.Errorf("failed to append to known_hosts file: %w", err)
		}
	}

	return nil
}

func spareSpace(target *os.File, totalSize int64, startAt int64, endAt int64, requiredSpace int64) error {
	lengthDiff := requiredSpace - (endAt - startAt)
	if lengthDiff == 0 {
		// No need to process
		return nil
	}

	// else: we have a job now
	buf := make([]byte, DefaultBufferSize)

	if lengthDiff > 0 {
		// Longer, move from back to front
		for end := totalSize + lengthDiff; end > endAt; {
			// Read
			start := end - DefaultBufferSize
			if start < endAt+lengthDiff {
				start = endAt + lengthDiff
			}
			readCount, err := target.ReadAt(buf, start)
			if err != nil {
				return fmt.Errorf("failed to read from file: %w", err)
			}

			// Write
			writeCount, err := target.WriteAt(buf[:readCount], start+lengthDiff)
			if err != nil {
				return fmt.Errorf("failed to write to file: %w", err)
			}

			if writeCount != readCount {
				return fmt.Errorf("read write mismatch, data corrupted")
			}

			// Update end
			end -= int64(readCount)
		}
	} else {
		// Shorter, move from front to back
		for start := startAt; start < totalSize+lengthDiff; {
			// Read
			readCount, err := target.ReadAt(buf, start)
			if err != nil {
				return fmt.Errorf("failed to read from file: %w", err)
			}

			if start+int64(readCount) > endAt+lengthDiff {
				readCount = int(endAt + lengthDiff - start)
			}

			// Write
			writeCount, err := target.WriteAt(buf[:readCount], start+lengthDiff)
			if err != nil {
				return fmt.Errorf("failed to write to file: %w", err)
			}

			if writeCount != readCount {
				return fmt.Errorf("read write mismatch, data corrupted")
			}

			// Update start
			start += int64(readCount)
		}

		// Truncate file after move to remove unexpected bytes
		if err := target.Truncate(totalSize + lengthDiff); err != nil {
			return fmt.Errorf("failed to truncate file: %w", err)
		}
	}

	return nil
}
