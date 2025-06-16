package types

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/GPTx-global/guru/app"
	"github.com/GPTx-global/guru/encoding"
	"github.com/GPTx-global/guru/oracle/config"
	"github.com/GPTx-global/guru/oracle/log"
	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

type JobType byte

const (
	Register JobType = iota
	Update
	Complete
)

type Job struct {
	ID     uint64
	URL    string
	Path   string
	Type   JobType
	Nonce  uint64
	Delay  time.Duration
	Status string
}

// MakeJobs creates a Job instance from various event types (OracleRequestDoc or blockchain events)
func MakeJobs(event any) []*Job {
	log.Debugf("start making jobs")
	jobs := make([]*Job, 0)

	switch event := event.(type) {
	case *oracletypes.OracleRequestDoc:
		myIndex := slices.Index(event.AccountList, config.Config.Address())
		index := (myIndex + 1) % len(event.Endpoints)
		jobs = append(jobs, &Job{
			ID:     event.RequestId,
			URL:    event.Endpoints[index].Url,
			Path:   event.Endpoints[index].ParseRule,
			Type:   Register,
			Nonce:  event.Nonce,
			Delay:  time.Duration(event.Period) * time.Second,
			Status: event.Status.String(),
		})
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
					myIndex := slices.Index(oracleMsg.RequestDoc.AccountList, config.Config.Address())
					index := (myIndex + 1) % len(oracleMsg.RequestDoc.Endpoints)
					requestID, err := strconv.ParseUint(event.Events[oracletypes.EventTypeRegisterOracleRequestDoc+"."+oracletypes.AttributeKeyRequestId][0], 10, 64)
					if err != nil {
						return nil
					}
					jobs = append(jobs, &Job{
						ID:     requestID,
						URL:    oracleMsg.RequestDoc.Endpoints[index].Url,
						Path:   oracleMsg.RequestDoc.Endpoints[index].ParseRule,
						Type:   Register,
						Nonce:  0,
						Delay:  time.Duration(oracleMsg.RequestDoc.Period) * time.Second,
						Status: oracleMsg.RequestDoc.Status.String(),
					})
				case *oracletypes.MsgUpdateOracleRequestDoc:
					fmt.Println("TODO: MsgUpdateOracleRequestDoc")
					jobs = nil
				}
			}
		case tmtypes.EventDataNewBlock:
			jobs = make([]*Job, len(event.Events[oracletypes.EventTypeCompleteOracleDataSet+"."+oracletypes.AttributeKeyRequestId]))
			for i := range jobs {
				jobs[i] = &Job{
					Type: Complete,
				}
			}

			for attributeKey, attributeValues := range event.Events {
				if strings.Contains(attributeKey, oracletypes.EventTypeCompleteOracleDataSet) {
					for i, attributeValue := range attributeValues {
						if strings.Contains(attributeKey, oracletypes.AttributeKeyRequestId) {
							jobs[i].ID, _ = strconv.ParseUint(attributeValue, 10, 64)
						} else if strings.Contains(attributeKey, oracletypes.AttributeKeyNonce) {
							jobs[i].Nonce, _ = strconv.ParseUint(attributeValue, 10, 64)
						}
					}
				}
			}
		}
	default:
		jobs = nil
	}

	if 0 < len(jobs) {
		for _, job := range jobs {
			log.Debugf("ID: %+v, Nonce: %+v", job.ID, job.Nonce)
		}
	}

	log.Debugf("end making jobs, %d", len(jobs))

	return jobs
}

type JobResult struct {
	ID    uint64
	Data  string
	Nonce uint64
}
