package types

import (
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/GPTx-global/guru/app"
	"github.com/GPTx-global/guru/encoding"
	"github.com/GPTx-global/guru/oracle/config"
	"github.com/GPTx-global/guru/oracle/log"
	feemarkettypes "github.com/GPTx-global/guru/x/feemarket/types"
	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

type Job struct {
	ID       uint64
	URL      string
	Path     string
	Type     JobType
	Nonce    uint64
	Delay    time.Duration
	Status   string
	GasPrice string
}

type JobResult struct {
	ID    uint64
	Data  string
	Nonce uint64
}

type JobType byte

const (
	Register JobType = iota
	Update
	Complete
	GasPrice
)

var (
	registerID          = oracletypes.EventTypeRegisterOracleRequestDoc + "." + oracletypes.AttributeKeyRequestId
	registerAccountList = oracletypes.EventTypeRegisterOracleRequestDoc + "." + oracletypes.AttributeKeyAccountList
	registerEndpoints   = oracletypes.EventTypeRegisterOracleRequestDoc + "." + oracletypes.AttributeKeyEndpoints
	registerPeriod      = oracletypes.EventTypeRegisterOracleRequestDoc + "." + oracletypes.AttributeKeyPeriod
	registerStatus      = oracletypes.EventTypeRegisterOracleRequestDoc + "." + oracletypes.AttributeKeyStatus

	updateID          = oracletypes.EventTypeUpdateOracleRequestDoc + "." + oracletypes.AttributeKeyRequestId
	updateAccountList = oracletypes.EventTypeUpdateOracleRequestDoc + "." + oracletypes.AttributeKeyAccountList
	updateEndpoints   = oracletypes.EventTypeUpdateOracleRequestDoc + "." + oracletypes.AttributeKeyEndpoints
	updatePeriod      = oracletypes.EventTypeUpdateOracleRequestDoc + "." + oracletypes.AttributeKeyPeriod
	updateStatus      = oracletypes.EventTypeUpdateOracleRequestDoc + "." + oracletypes.AttributeKeyStatus

	completeID    = oracletypes.EventTypeCompleteOracleDataSet + "." + oracletypes.AttributeKeyRequestId
	completeNonce = oracletypes.EventTypeCompleteOracleDataSet + "." + oracletypes.AttributeKeyNonce

	minGasPrice = feemarkettypes.EventTypeChangeMinGasPrice + "." + feemarkettypes.AttributeKeyMinGasPrice
)

func MakeJobs(event coretypes.ResultEvent) []*Job {
	// if _, ok := event.Data.(tmtypes.EventDataNewBlock); !ok {
	// 	return nil
	// }

	log.Debugf("start making jobs")

	jobs := make([]*Job, 0)
	eventsMap := event.Events

	if _, ok := eventsMap[updateID]; ok {
		jobs = append(jobs, makeUpdateJobs(eventsMap)...)
	}
	if _, ok := eventsMap[registerID]; ok {
		jobs = append(jobs, makeRegisterJobs(eventsMap)...)
	}
	if _, ok := eventsMap[minGasPrice]; ok {
		jobs = append(jobs, makeMinGasPriceJobs(eventsMap)...)
	}
	if _, ok := eventsMap[completeID]; ok {
		jobs = append(jobs, makeCompleteJobs(eventsMap)...)
	}

	log.Debugf("end making jobs, %d", len(jobs))

	if len(jobs) == 0 {
		return nil
	}

	return jobs
}

func makeRegisterJobs(eventMap map[string][]string) []*Job {
	jobs := make([]*Job, 0)

	for i, id := range eventMap[registerID] {
		if !validateAddress(eventMap[registerAccountList][i]) {
			log.Debugf("register job %s is not for me", id)
			continue
		}

		requestID, err := strconv.ParseUint(id, 10, 64)
		if err != nil {
			log.Errorf("failed to parse requestID: %v", err)
			continue
		}

		endpoints := []*oracletypes.OracleEndpoint{}
		err = json.Unmarshal([]byte(eventMap[registerEndpoints][i]), &endpoints)
		if err != nil {
			log.Errorf("failed to unmarshal endpoints: %v", err)
			continue
		}

		myIndex := slices.Index(eventMap[registerAccountList], config.Address().String())
		index := (myIndex + 1) % len(endpoints)

		period, err := strconv.ParseUint(eventMap[registerPeriod][i], 10, 64)
		if err != nil {
			log.Errorf("failed to parse period: %v", err)
			continue
		}

		jobs = append(jobs, &Job{
			ID:     requestID,
			URL:    endpoints[index].Url,
			Path:   endpoints[index].ParseRule,
			Type:   Register,
			Nonce:  0,
			Delay:  time.Duration(period) * time.Second,
			Status: eventMap[registerStatus][i],
		})
	}

	return jobs
}

func makeUpdateJobs(eventMap map[string][]string) []*Job {
	jobs := make([]*Job, 0)

	for i, id := range eventMap[updateID] {
		if !validateAddress(eventMap[updateAccountList][i]) {
			log.Debugf("update job %s is not for me", id)
			continue
		}

		requestID, err := strconv.ParseUint(id, 10, 64)
		if err != nil {
			log.Errorf("failed to parse requestID: %v", err)
			continue
		}

		endpoints := []*oracletypes.OracleEndpoint{}
		err = json.Unmarshal([]byte(eventMap[updateEndpoints][i]), &endpoints)
		if err != nil {
			log.Errorf("failed to unmarshal endpoints: %v", err)
			continue
		}

		myIndex := slices.Index(eventMap[updateAccountList], config.Address().String())
		index := (myIndex + 1) % len(endpoints)

		period, err := strconv.ParseUint(eventMap[updatePeriod][i], 10, 64)
		if err != nil {
			log.Errorf("failed to parse period: %v", err)
			continue
		}

		jobs = append(jobs, &Job{
			ID:     requestID,
			URL:    endpoints[index].Url,
			Path:   endpoints[index].ParseRule,
			Type:   Update,
			Delay:  time.Duration(period) * time.Second,
			Status: eventMap[updateStatus][i],
		})
	}

	return jobs
}

func makeCompleteJobs(eventMap map[string][]string) []*Job {
	jobs := make([]*Job, 0)

	for i, id := range eventMap[completeID] {
		requestID, err := strconv.ParseUint(id, 10, 64)
		if err != nil {
			log.Errorf("failed to parse requestID: %v", err)
			continue
		}

		nonce, err := strconv.ParseUint(eventMap[completeNonce][i], 10, 64)
		if err != nil {
			log.Errorf("failed to parse nonce: %v", err)
			continue
		}

		jobs = append(jobs, &Job{
			ID:    requestID,
			Nonce: nonce,
			Type:  Complete,
		})
	}

	return jobs
}

func makeMinGasPriceJobs(eventMap map[string][]string) []*Job {
	jobs := make([]*Job, 0)

	for _, gasPrice := range eventMap[minGasPrice] {
		jobs = append(jobs, &Job{
			Type:     GasPrice,
			GasPrice: gasPrice,
		})
	}

	return jobs
}

func validateAddress(accountList string) bool {
	return strings.Contains(accountList, config.Address().String())
}

// MakeJob creates a Job instance from various event types (OracleRequestDoc or blockchain events)
func MakeJob(event any) []*Job {
	log.Debugf("start making jobs")
	jobs := make([]*Job, 0)

	switch event := event.(type) {
	case *oracletypes.OracleRequestDoc:
		myIndex := slices.Index(event.AccountList, config.Address().String())
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
					myIndex := slices.Index(oracleMsg.RequestDoc.AccountList, config.Address().String())
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
					jobs = nil
				}
			}
			if len(jobs) == 0 {
				return nil
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
		default:
			return nil
		}
	}

	if 0 < len(jobs) {
		for _, job := range jobs {
			log.Debugf("ID: %+v, Nonce: %+v", job.ID, job.Nonce)
		}
	}

	log.Debugf("end making jobs, %d", len(jobs))

	return jobs
}
