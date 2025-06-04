// x/mymodule/keeper/keeper.go
package keeper

import (
	"encoding/binary"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/log"

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
		k.cdc.MustUnmarshal(iterator.Value(), &data)
		datas = append(datas, &data)
	}
	return datas, nil
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
