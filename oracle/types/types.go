package types

import (
	"time"

	feemarkettypes "github.com/GPTx-global/guru/x/feemarket/types"
	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
)

var (
	RegisterID          = oracletypes.EventTypeRegisterOracleRequestDoc + "." + oracletypes.AttributeKeyRequestId
	RegisterAccountList = oracletypes.EventTypeRegisterOracleRequestDoc + "." + oracletypes.AttributeKeyAccountList

	UpdateID = oracletypes.EventTypeUpdateOracleRequestDoc + "." + oracletypes.AttributeKeyRequestId

	CompleteID    = oracletypes.EventTypeCompleteOracleDataSet + "." + oracletypes.AttributeKeyRequestId
	CompleteNonce = oracletypes.EventTypeCompleteOracleDataSet + "." + oracletypes.AttributeKeyNonce

	MinGasPrice = feemarkettypes.EventTypeChangeMinGasPrice + "." + feemarkettypes.AttributeKeyMinGasPrice
)

type Job struct {
	ID     uint64
	URL    string
	Path   string
	Nonce  uint64
	Delay  time.Duration
	Status oracletypes.RequestStatus
}

type JobResult struct {
	ID    uint64
	Data  string
	Nonce uint64
}
