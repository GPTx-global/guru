package types

import (
	"fmt"
	"strconv"
	"time"

	"github.com/GPTx-global/guru/app"
	"github.com/GPTx-global/guru/encoding"
	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

type Job struct {
	ID     uint64
	URL    string
	Path   string
	Nonce  uint64
	Delay  time.Duration
	Status string
}

// MakeJob creates a Job instance from various event types (OracleRequestDoc or blockchain events)
func MakeJob(event any) *Job {
	job := &Job{}

	switch event := event.(type) {
	case *oracletypes.OracleRequestDoc:
		job.ID = event.RequestId
		job.URL = event.Endpoints[0].Url
		job.Path = event.Endpoints[0].ParseRule
		job.Nonce = event.Nonce
		job.Delay = time.Duration(event.Period) * time.Second
		job.Status = event.Status.String()
	case coretypes.ResultEvent:
		switch event.Data.(type) {
		case tmtypes.EventDataTx:
			txDecoder := encoding.MakeConfig(app.ModuleBasics).TxConfig.TxDecoder()
			tx, err := txDecoder(event.Data.(tmtypes.EventDataTx).Tx)
			if err != nil {
				return nil
			}

			msgs := tx.GetMsgs()
			for _, msg := range msgs {
				switch oracleMsg := msg.(type) {
				case *oracletypes.MsgRegisterOracleRequestDoc:
					job.ID = oracleMsg.RequestDoc.RequestId
					job.URL = oracleMsg.RequestDoc.Endpoints[0].Url
					job.Path = oracleMsg.RequestDoc.Endpoints[0].ParseRule
					job.Nonce = oracleMsg.RequestDoc.Nonce
					job.Delay = time.Duration(oracleMsg.RequestDoc.Period) * time.Second
					job.Status = oracleMsg.RequestDoc.Status.String()
				case *oracletypes.MsgUpdateOracleRequestDoc:
					fmt.Println("TODO: MsgUpdateOracleRequestDoc")
					job = nil
				}
			}
		case tmtypes.EventDataNewBlock:
			requestID, err := strconv.ParseUint(event.Events[oracletypes.EventTypeCompleteOracleDataSet+"."+oracletypes.AttributeKeyRequestId][0], 10, 64)
			if err != nil {
				return nil
			}

			nonce, err := strconv.ParseUint(event.Events[oracletypes.EventTypeCompleteOracleDataSet+"."+oracletypes.AttributeKeyNonce][0], 10, 64)
			if err != nil {
				return nil
			}

			job.ID = requestID
			job.Nonce = nonce
		}
	default:
		job = nil
	}

	fmt.Printf("job: %+v\n", job)

	return job
}

type JobResult struct {
	ID    uint64
	Data  string
	Nonce uint64
}
