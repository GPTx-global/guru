package types

import "time"

type Job struct {
	ID    string
	URL   string
	Path  string
	Nonce uint64
	Delay time.Duration
}

type OracleData struct {
	RequestID string
	Data      string
	Nonce     uint64
}
