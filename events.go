package main

import (
	"encoding/json"
	"fmt"
)

const (
	EventTransmitStart     = '\x02' // ASCII: Start of Text
	EventTransmitEnd       = '\x03' // ASCII: End of Text
	EventTransmitSeparator = '\x1f' // ASCII: Unit Separator
)

const (
	EventNameServerKeyNew     = "serverKeyNew"     // new server, never seen before
	EventNameServerKeyChanged = "serverKeyChanged" // old server with new key
	EventNameSSHStart         = "sshStart"         // pipe stdin/stdout/stderr to ssh from now on
)

type EventPayloadHostKeyNew struct {
	Fingerprint string `json:"fp"`
}

type EventPayloadHostKeyChanged struct {
	OldFingerprint string `json:"ofp"`
	Fingerprint    string `json:"fp"`
}

func buildEvent(name string, payload *any) ([]byte, error) {
	data := []byte{EventTransmitStart}
	data = append(data, name...)
	if payload != nil {
		data = append(data, EventTransmitSeparator)
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		data = append(data, payloadBytes...)
	}
	data = append(data, EventTransmitEnd)
	return data, nil
}
