package main

import (
	"bytes"
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

		dataBuf := inBuf[:n]

		for { // Process all events in a single input event (latest one overwrites all before)
			// Check if event exists
			dataLen := len(dataBuf)
			eventStartIndex := bytes.Index(dataBuf, EscapeWindowChangePrefix)

			if eventStartIndex == -1 {
				// No event, just pipe normally
				_, err = to.Write(dataBuf[:dataLen])
				if err != nil {
					return fmt.Errorf("failed to pipe to stdin: %w", err)
				}
				break // Proceed to next loop
			} else if eventStartIndex > 0 { // 0 means nothing to send
				// Send bytes before event
				_, err = to.Write(dataBuf[:eventStartIndex])
				if err != nil {
					return fmt.Errorf("failed to pipe to stdin: %w", err)
				}
			}

			// Match suffix
			eventEndIndex := eventStartIndex + len(EscapeWindowChangePrefix)

			// else: prefix all match, extract all bytes into event buffer till match suffix
			for ; eventEndIndex < dataLen; eventEndIndex++ {
				if dataBuf[eventEndIndex] == EscapeWindowChangeSuffix {
					break
				}
			}

			if eventEndIndex >= dataLen {
				// Incomplete event, just pipe normally
				_, err = to.Write(dataBuf[eventStartIndex:])
				if err != nil {
					return fmt.Errorf("failed to pipe to stdin: %w", err)
				}
				break // Proceed to next loop
			}

			// else: event all extracted! time to analyse
			// Maybe contain some incomplete event fragments, so go reversely to find if any prefix match
			eventPayload := dataBuf[eventStartIndex:eventEndIndex] // discard suffix
			eventPrefixLastIndex := bytes.LastIndex(eventPayload, EscapeWindowChangePrefix)
			if eventPrefixLastIndex > 0 {
				// Include invalid events

				// Send invalid events as raw content
				_, err = to.Write(eventPayload[:eventPrefixLastIndex])
				if err != nil {
					return fmt.Errorf("failed to pipe to stdin: %w", err)
				}
			}

			// Process event
			if err = procWindowChangeEvent(string(eventPayload[eventPrefixLastIndex+len(EscapeWindowChangePrefix):]), windowResize); err != nil {
				// Something is wrong, we can't handle this event, so send without processing
				_, err = to.Write(dataBuf[eventStartIndex+eventPrefixLastIndex:]) // Send everything
				if err != nil {
					return fmt.Errorf("failed to proc window change event: %w", err)
				}
				break // Proceed to next loop
			}

			// Continue to process remain bytes
			dataBuf = dataBuf[eventEndIndex+1:]
		}
	}

	return nil
}
