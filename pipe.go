package main

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func pipe(from io.Reader, to io.Writer) error {
	buf := make([]byte, DefaultBufferSize)
	for {
		n, err := from.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			} else {
				return fmt.Errorf("failed to read from reader: %w", err)
			}
		}

		if _, err := to.Write(buf[:n]); err != nil {
			return fmt.Errorf("failed to write to writer: %w", err)
		}
	}

	return nil
}

func procWindowChangeEvent(eventPayload string, windowResize func(h int, w int) error) error {
	splits := strings.Split(eventPayload, ";")
	if len(splits) != 2 {
		return fmt.Errorf("invalid event payload: %s", eventPayload)
	}

	rows, err := strconv.Atoi(splits[0])
	if err != nil {
		return fmt.Errorf("failed to parse rows: %w", err)
	}

	cols, err := strconv.Atoi(splits[1])
	if err != nil {
		return fmt.Errorf("failed to parse cols: %w", err)
	}

	// Send new rows & cols to server
	err = windowResize(rows, cols)
	if err != nil {
		return fmt.Errorf("failed to set window size: %w", err)
	}

	// No error
	return nil
}

func inPipe(from io.Reader, to io.Writer, windowResize func(h int, w int) error) error {
	inBuf := make([]byte, DefaultBufferSize)
	eventBuf := make([]byte, DefaultBufferSize)

	evSize := 0

	for {
		// Receive data
		n, err := from.Read(inBuf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			} else {
				return fmt.Errorf("failed to read from stdin: %w", err)
			}
		}

		// Set start index
		start := 0
		if evSize > 0 {
			// Last loop remains unprocessed data, continue to process
			if evSize < EscapeWindowChangePrefixLen {
				// Prefix not matched, start to match prefix
				for ; start+evSize <= EscapeWindowChangePrefixLen; start++ { // Start from the first byte after \x1B
					eventBuf[start+evSize] = inBuf[start]
					if inBuf[start] != EscapeWindowChangePrefix[start+evSize-1] {
						// Mismatch
						start = 0
						break
					}
				}

				if start == 0 { // Mismatch
					// Unable to handle this, send buffered event data to server
					_, err = to.Write(eventBuf[:evSize])
					if err != nil {
						return fmt.Errorf("failed to pipe to stdin: %w", err)
					}
					evSize = 0 // reset
				}
			}

			if evSize > 0 {
				// Prefix match, start receiving event bytes till match suffix
				for ; start < n; start++ {
					eventBuf[start+evSize] = inBuf[start]
					if inBuf[start] == EscapeWindowChangeSuffix {
						break
					}
				}

				if start >= n {
					// Message incomplete, save it and wait for next loop
					evSize += start
					continue
				}

				// else: suffix match, message complete
				eventPayload := string(eventBuf[EscapeWindowChangePrefixLen : evSize+start]) // discard prefix and suffix
				err = procWindowChangeEvent(eventPayload, windowResize)
				if err != nil {
					// Unable to handle this, send buffered event data to server
					_, err = to.Write(eventBuf[:evSize])
					if err != nil {
						return fmt.Errorf("failed to pipe to stdin: %w", err)
					}
					start = 0 // reset
				} else {
					start += 1 // Skip suffix
				}

				evSize = 0 // reset
			}
		}

		i := start

		// Check byte by byte
		for ; i < n; i++ {
			if inBuf[i] == EscapeWindowChangePrefix[0] { // ESC for ANSI escape sequences
				break
			}
		}

		if i >= n {
			// No event, just pipe normally
			_, err = to.Write(inBuf[start:n])
			if err != nil {
				return fmt.Errorf("failed to pipe to stdin: %w", err)
			}
			continue // Proceed to next loop
		}

		// Otherwise: event caused early end
		_, err = to.Write(inBuf[start:i]) // pipe data before event
		if err != nil {
			return fmt.Errorf("failed to pipe to stdin: %w", err)
		}

		// Match prefix
		for evSize = 0; (i+evSize < n) && (evSize < EscapeWindowChangePrefixLen); evSize++ { // Start from the first byte
			eventBuf[evSize] = inBuf[i+evSize]
			if inBuf[i+evSize] != EscapeWindowChangePrefix[evSize] {
				// Mismatch
				evSize = 0
				break
			}
		}

		if evSize == 0 { // Mismatch
			// Unable to handle this, send to server
			_, err = to.Write(inBuf[i:n])
			if err != nil {
				return fmt.Errorf("failed to pipe to stdin: %w", err)
			}
			continue // Proceed to next loop
		}

		if i+evSize >= n {
			// Message incomplete, save it and wait for next loop
			continue
		}

		// else: prefix all match, extract all bytes into event buffer till match suffix
		for ; i+evSize < n; evSize++ {
			eventBuf[evSize] = inBuf[i+evSize]
			if inBuf[i+evSize] == EscapeWindowChangeSuffix {
				break
			}
		}

		if i+evSize >= n {
			// Message incomplete, save it and wait for next loop
			continue
		}

		// else: event all extracted! time to analyse
		eventPayload := string(eventBuf[EscapeWindowChangePrefixLen:evSize]) // discard prefix and suffix
		err = procWindowChangeEvent(eventPayload, windowResize)
		if err != nil {
			// Something is wrong, we can't handle this event, so send without processing
			evSize = 0                    // reset
			_, err = to.Write(inBuf[i:n]) // Send everything
			if err != nil {
				return fmt.Errorf("failed to proc window change event: %w", err)
			}
			continue // Proceed to next loop
		}

		// Send remain bytes
		_, err = to.Write(inBuf[i+evSize+1 : n])
		if err != nil {
			return fmt.Errorf("failed to pipe to stdin: %w", err)
		}

		// Reset event size as already processed
		evSize = 0
	}

	return nil
}
