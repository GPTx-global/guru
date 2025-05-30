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

	accountKeeper types.AccountKeeper
	bankKeeper    types.BankKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
) *Keeper {
	return &Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		accountKeeper: accountKeeper,
		bankKeeper:    bankKeeper,
	}
}

func (k Keeper) SetOracleRequestDocCount(ctx sdk.Context, count uint64) {
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(types.KeyOracleRequestDocCount, bz)
}

func (k Keeper) GetOracleRequestDocCount(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyOracleRequestDocCount)
	if len(bz) == 0 {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

func (k Keeper) SetOracleRequestDoc(ctx sdk.Context, doc types.RequestOracleDoc) {
	store := ctx.KVStore(k.storeKey)

	bz := k.cdc.MustMarshal(&doc)
	store.Set(types.GetOracleRequestDocKey(doc.RequestId), bz)
}

func (k Keeper) GetOracleRequestDoc(ctx sdk.Context, id uint64) (*types.RequestOracleDoc, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetOracleRequestDocKey(id))
	if len(bz) == 0 {
		return nil, fmt.Errorf("Not exist RequestDoc(req_id: %d)", id)
	}

	var doc types.RequestOracleDoc
	k.cdc.MustUnmarshal(bz, &doc)
	return &doc, nil
}

// GetModeratorAddress returns the current moderator address.
func (k Keeper) GetModeratorAddress(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(types.KeyModeratorAddress))
	if len(bz) == 0 {
		return ""
	}
	return string(bz)
}

// SetModeratorAddress adds/updates the moderator address.
func (k Keeper) SetModeratorAddress(ctx sdk.Context, moderator_address string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyModeratorAddress, []byte(moderator_address))
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
