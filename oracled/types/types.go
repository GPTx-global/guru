package types

import "time"

type Job struct {
	ID          uint64        // Oracle Request ID
	URL         string        // Data source URL
	Path        string        // JSON path for data extraction
	Nonce       uint64        // Current nonce
	Delay       time.Duration // Update interval
	MessageType string        // "register" or "update"
}

type OracleData struct {
	RequestID uint64
	Data      string
	Nonce     uint64
}
