// x/mymodule/keeper/keeper.go
package keeper

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/log"

	errorsmod "cosmossdk.io/errors"
	"github.com/GPTx-global/guru/x/oracle/types"
)

type Keeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey
	// hooks    types.OracleHooks
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
) *Keeper {
	return &Keeper{
		cdc:      cdc,
		storeKey: storeKey,
	}
}

// SetParams stores the oracle module parameters in the state store
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}

	store.Set(types.KeyParams, bz)

	return nil
}

// GetParams retrieves the oracle module parameters from the state store
// Returns default parameters if no parameters are found
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyParams)
	if len(bz) == 0 {
		return types.DefaultParams()
	}

	var params types.Params
	k.cdc.MustUnmarshal(bz, &params)
	return params
}

// SetModeratorAddress stores the moderator address in the state store
func (k Keeper) SetModeratorAddress(ctx sdk.Context, address string) error {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyModeratorAddress, []byte(address))
	return nil
}

// GetModeratorAddress retrieves the moderator address from the state store
// Returns empty string if no address is found
func (k Keeper) GetModeratorAddress(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyModeratorAddress)
	if len(bz) == 0 {
		return ""
	}
	return string(bz)
}

// SetOracleRequestDocCount stores the total count of oracle request documents in the state store
// count: number of documents to store
func (k Keeper) SetOracleRequestDocCount(ctx sdk.Context, count uint64) {
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(types.KeyOracleRequestDocCount, bz)
}

// GetOracleRequestDocCount retrieves the total count of oracle request documents from the state store
// Returns: number of stored documents (0 if none exist)
func (k Keeper) GetOracleRequestDocCount(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyOracleRequestDocCount)
	if len(bz) == 0 {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// SetOracleRequestDoc stores an oracle request document in the state store
// doc: oracle request document to store
func (k Keeper) SetOracleRequestDoc(ctx sdk.Context, doc types.OracleRequestDoc) {
	store := ctx.KVStore(k.storeKey)

	bz := k.cdc.MustMarshal(&doc)
	store.Set(types.GetOracleRequestDocKey(doc.RequestId), bz)
}

// GetOracleRequestDoc retrieves an oracle request document by ID from the state store
// id: ID of the document to retrieve
// Returns: retrieved oracle request document and error (error if document doesn't exist)
func (k Keeper) GetOracleRequestDoc(ctx sdk.Context, id uint64) (*types.OracleRequestDoc, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetOracleRequestDocKey(id))
	if len(bz) == 0 {
		return nil, fmt.Errorf("not exist RequestDoc(req_id: %d)", id)
	}

	var doc types.OracleRequestDoc
	k.cdc.MustUnmarshal(bz, &doc)
	return &doc, nil
}

func (k Keeper) GetOracleRequestDocs(ctx sdk.Context) []*types.OracleRequestDoc {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, types.KeyOracleRequestDoc)
	defer iterator.Close()

	var docs []*types.OracleRequestDoc
	for ; iterator.Valid(); iterator.Next() {
		var doc types.OracleRequestDoc
		k.cdc.MustUnmarshal(iterator.Value(), &doc)
		docs = append(docs, &doc)
	}
	return docs
}

func (k Keeper) SetSubmitData(ctx sdk.Context, data types.SubmitDataSet) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&data)
	key := types.GetSubmitDataKeyByProvider(data.RequestId, data.Nonce, data.Provider)
	store.Set(key, bz)
}

func (k Keeper) GetSubmitData(ctx sdk.Context, requestId uint64, nonce uint64, provider string) ([]*types.SubmitDataSet, error) {
	store := ctx.KVStore(k.storeKey)
	var datas []*types.SubmitDataSet
	if provider == "" {
		datas, err := k.GetSubmitDatas(ctx, requestId, nonce)
		if err != nil {
			return nil, err
		}
		return datas, nil
	} else {
		key := types.GetSubmitDataKeyByProvider(requestId, nonce, provider)
		bz := store.Get(key)
		if len(bz) == 0 {
			return nil, fmt.Errorf("not exist SubmitData(req_id: %d, nonce: %d, provider: %s)", requestId, nonce, provider)
		}

		var data types.SubmitDataSet
		k.cdc.MustUnmarshal(bz, &data)
		datas = append(datas, &data)
	}
	return datas, nil
}

func (k Keeper) GetSubmitDatas(ctx sdk.Context, requestId uint64, nonce uint64) ([]*types.SubmitDataSet, error) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, types.GetSubmitDataKey(requestId, nonce))
	defer iterator.Close()

	var datas []*types.SubmitDataSet
	for ; iterator.Valid(); iterator.Next() {
		var data types.SubmitDataSet
		if err := k.cdc.Unmarshal(iterator.Value(), &data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal submit data: %w", err)
		}
		datas = append(datas, &data)
	}
	return datas, nil
}

// SetDataSet stores the aggregated oracle data
func (k Keeper) SetDataSet(ctx sdk.Context, dataSet types.DataSet) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&dataSet)
	store.Set(types.GetDataSetKey(dataSet.RequestId, dataSet.Nonce), bz)
}

func (k Keeper) GetDataSet(ctx sdk.Context, requestId uint64, nonce uint64) (*types.DataSet, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetDataSetKey(requestId, nonce))
	if len(bz) == 0 {
		return nil, fmt.Errorf("not exist DataSet(req_id: %d, nonce: %d)", requestId, nonce)
	}

	var dataSet types.DataSet
	k.cdc.MustUnmarshal(bz, &dataSet)
	return &dataSet, nil
}

// Logger returns a logger instance with the module name prefixed
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) checkAccountAuthorized(accountList []string, fromAddress string) bool {
	for _, account := range accountList {
		if account == fromAddress {
			return true
		}
	}
	return false
}

// ProcessOracleDataSetAggregation processes oracle data set aggregation for all enabled requests
// It performs the following steps:
// 1. Retrieves all registered OracleRequestDocs
// 2. For each enabled document:
//   - Gets submit data sets for the next nonce
//   - Checks if quorum is met
//   - Aggregates data based on the rule
//   - Stores the result and emits events
func (k Keeper) ProcessOracleDataSetAggregation(ctx sdk.Context) {
	// Get all registered OracleRequestDocs
	requestDocs := k.GetOracleRequestDocs(ctx)
	if len(requestDocs) == 0 {
		k.Logger(ctx).Info("no oracle request docs found")
		return
	}

	// Process each OracleRequestDoc
	for _, doc := range requestDocs {
		if doc.Status != types.RequestStatus_REQUEST_STATUS_ENABLED {
			continue
		}

		// Get submit data sets for current request_id and next nonce
		nextNonce := doc.Nonce + 1
		submitDatas, err := k.GetSubmitDatas(ctx, doc.RequestId, nextNonce)
		if err != nil {
			k.Logger(ctx).Error("failed to get submit datas",
				"request_id", doc.RequestId,
				"nonce", nextNonce,
				"error", err)
			continue
		}

		// Check if we have enough submissions (quorum)
		if uint32(len(submitDatas)) < doc.Quorum {
			k.Logger(ctx).Info(fmt.Sprintf("insufficient submissions for request_id %d, nonce %d: got %d, need %d",
				doc.RequestId, nextNonce, len(submitDatas), doc.Quorum))
			continue
		}

		// Aggregate data based on AggregationRule
		aggregatedValue, err := k.aggregateData(doc.AggregationRule, submitDatas)
		if err != nil {
			k.Logger(ctx).Error(fmt.Sprintf("failed to aggregate data for request_id %d: %v",
				doc.RequestId, err))
			continue
		}

		// Create and store DataSet
		dataSet := types.DataSet{
			RequestId:   doc.RequestId,
			Nonce:       nextNonce,
			BlockHeight: uint64(ctx.BlockHeight()),
			BlockTime:   uint64(ctx.BlockTime().Unix()),
			RawData:     aggregatedValue,
		}
		k.SetDataSet(ctx, dataSet)

		// Increment nonce
		doc.Nonce = nextNonce
		k.SetOracleRequestDoc(ctx, *doc)

		// Emit event
		ctx.EventManager().EmitEvents(
			sdk.Events{
				sdk.NewEvent(
					types.EventTypeCompleteOracleDataSet,
					sdk.NewAttribute(types.AttributeKeyRequestId, fmt.Sprintf("%d", doc.RequestId)),
					sdk.NewAttribute(types.AttributeKeyNonce, fmt.Sprintf("%d", nextNonce)),
					sdk.NewAttribute(types.AttributeKeyRawData, aggregatedValue),
				),
			},
		)

		k.Logger(ctx).Info(fmt.Sprintf("successfully processed oracle request %d with nonce %d",
			doc.RequestId, nextNonce))
	}
}

// aggregateData aggregates the submitted data based on the aggregation rule
func (k Keeper) aggregateData(rule types.AggregationRule, submitDatas []*types.SubmitDataSet) (string, error) {
	switch rule {
	case types.AggregationRule_AGGREGATION_RULE_AVG:
		return k.calculateAverage(submitDatas)
	case types.AggregationRule_AGGREGATION_RULE_MIN:
		return k.calculateMin(submitDatas)
	case types.AggregationRule_AGGREGATION_RULE_MAX:
		return k.calculateMax(submitDatas)
	case types.AggregationRule_AGGREGATION_RULE_MEDIAN:
		return k.calculateMedian(submitDatas)
	default:
		return "", fmt.Errorf("unsupported aggregation rule: %s", rule)
	}
}

func (k Keeper) calculateMax(submitDatas []*types.SubmitDataSet) (string, error) {
	if len(submitDatas) == 0 {
		return "", fmt.Errorf("no data to calculate max")
	}

	max := new(big.Float)
	for _, data := range submitDatas {
		value := new(big.Float)
		value.SetString(data.RawData)
		if value.Cmp(max) > 0 {
			max = value
		}
	}
	return max.Text('f', -1), nil
}

func (k Keeper) calculateMin(submitDatas []*types.SubmitDataSet) (string, error) {
	if len(submitDatas) == 0 {
		return "", fmt.Errorf("no data to calculate min")
	}

	min := new(big.Float)
	for _, data := range submitDatas {
		value := new(big.Float)
		value.SetString(data.RawData)
		if value.Cmp(min) < 0 {
			min = value
		}
	}
	return min.Text('f', -1), nil
}

// calculateAverage calculates the average of all submitted values
func (k Keeper) calculateAverage(submitDatas []*types.SubmitDataSet) (string, error) {
	if len(submitDatas) == 0 {
		return "", fmt.Errorf("no data to average")
	}

	sum := new(big.Float)
	for _, data := range submitDatas {
		value := new(big.Float)
		value.SetString(data.RawData)
		sum.Add(sum, value)
	}

	avg := new(big.Float).Quo(sum, new(big.Float).SetInt64(int64(len(submitDatas))))
	return avg.Text('f', -1), nil
}

// calculateMedian calculates the median of all submitted values
func (k Keeper) calculateMedian(submitDatas []*types.SubmitDataSet) (string, error) {
	if len(submitDatas) == 0 {
		return "", fmt.Errorf("no data to calculate median")
	}

	values := make([]*big.Float, len(submitDatas))
	for i, data := range submitDatas {
		values[i] = new(big.Float)
		values[i].SetString(data.RawData)
	}

	// Sort values
	for i := 0; i < len(values)-1; i++ {
		for j := i + 1; j < len(values); j++ {
			if values[i].Cmp(values[j]) > 0 {
				values[i], values[j] = values[j], values[i]
			}
		}
	}

	// Calculate median
	mid := len(values) / 2
	if len(values)%2 == 0 {
		// Even number of values, average the middle two
		median := new(big.Float).Add(values[mid-1], values[mid])
		median.Quo(median, new(big.Float).SetInt64(2))
		return median.Text('f', -1), nil
	}
	// Odd number of values, return the middle one
	return values[mid].Text('f', -1), nil
}

func (k Keeper) validateSubmitData(data types.SubmitDataSet) error {
	if data.RequestId == 0 {
		return errorsmod.Wrapf(types.ErrInvalidRequestId, "request id is 0")
	}
	if data.Nonce == 0 {
		return errorsmod.Wrapf(types.ErrInvalidNonce, "nonce is 0")
	}
	if data.Provider == "" {
		return errorsmod.Wrapf(types.ErrInvalidProvider, "provider is empty")
	}
	if data.RawData == "" {
		return errorsmod.Wrapf(types.ErrInvalidRawData, "raw data is empty")
	}
	return nil
}
