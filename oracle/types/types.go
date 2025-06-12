package types

import (
	"time"

	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

type Job struct {
	ID     uint64
	URL    string
	Path   string
	Nonce  uint64
	Delay  time.Duration
	Status string
}

func MakeJob(event any) *Job {
	job := new(Job)

	switch event := event.(type) {
	case *oracletypes.OracleRequestDoc:
		_ = event
	case coretypes.ResultEvent:
		_ = event
	default:
		job = nil
	}

	return job
}

type JobResult struct {
	ID    uint64
	Data  string
	Nonce uint64
}
