package keeper

import (
	"fmt"
	"strings"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/GPTx-global/guru/x/cex/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	transfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	"github.com/tendermint/tendermint/libs/log"
)

// Keeper of the xmsquare store
type Keeper struct {
	// Protobuf codec
	cdc codec.BinaryCodec

	// Store key required for the EVM Prefix KVStore. It is required by:
	// - storing account's Storage State
	// - storing account's Code
	// - storing transaction Logs
	// - storing Bloom filters by block height. Needed for the Web3 API.
	storeKey storetypes.StoreKey

	// mint and burn coins using bankkeper
	bankKeeper types.BankKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	key storetypes.StoreKey,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) Keeper {

	// ensure cex module account is set
	if addr := ak.GetModuleAddress(types.ModuleName); addr == nil {
		panic("the cex module account has not been set")
	}

	return Keeper{
		cdc:        cdc,
		storeKey:   key,
		bankKeeper: bk,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

func (k Keeper) SwapCoins(ctx sdk.Context, senderAcc sdk.AccAddress, fromDenom, fromChannel, toDenom, toChannel string, fromAmount math.Int) error {
	reserveAddr := k.GetReserveAccount(ctx)
	reserveAcc := sdk.MustAccAddressFromBech32(reserveAddr)
	fromIBCDenom := k.GetIBCDenom(fromDenom, fromChannel)
	toIBCDenom := k.GetIBCDenom(toDenom, toChannel)
	rate := k.GetRate(ctx, k.GetPairDenom(fromDenom, toDenom))
	toAmount := rate.MulInt(fromAmount).TruncateInt()

	senderBalance := k.bankKeeper.GetBalance(ctx, senderAcc, k.GetIBCDenom(fromDenom, fromChannel))

	if senderBalance.Amount.LT(fromAmount) {
		return errorsmod.Wrapf(types.ErrInsufficientBalance, ", %s is less than %s", senderBalance.String(), fromAmount.String()+fromDenom)
	}

	reserveBalance := k.bankKeeper.GetBalance(ctx, reserveAcc, k.GetIBCDenom(toDenom, toChannel))
	if reserveBalance.Amount.LT(toAmount) {
		return errorsmod.Wrapf(types.ErrInsufficientReserve, ", %s is less than %s", senderBalance.String(), fromAmount.String()+fromDenom)
	}

	err := k.bankKeeper.SendCoins(ctx, senderAcc, reserveAcc, sdk.NewCoins(sdk.NewCoin(fromIBCDenom, fromAmount)))
	if err != nil {
		return err
	}
	err = k.bankKeeper.SendCoins(ctx, reserveAcc, senderAcc, sdk.NewCoins(sdk.NewCoin(toIBCDenom, toAmount)))
	if err != nil {
		return err
	}
	return nil
}

// GetModeratorAddress returns the current moderator address.
func (k Keeper) GetModeratorAddress(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyModeratorAddress)
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

// GetReserveAccount returns the current reserve address.
func (k Keeper) GetReserveAccount(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyReserveAccount)
	if len(bz) == 0 {
		return ""
	}
	return string(bz)
}

// SetReserveAccount adds/updates the reserve address.
func (k Keeper) SetReserveAccount(ctx sdk.Context, reserve_address string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyReserveAccount, []byte(reserve_address))
}

func (k Keeper) GetReserve(ctx sdk.Context, denom string) sdk.Coin {
	reserveAddr := k.GetReserveAccount(ctx)

	return k.bankKeeper.GetBalance(ctx, sdk.MustAccAddressFromBech32(reserveAddr), denom)
}

func (k Keeper) SetRate(ctx sdk.Context, pairDenom string, rate sdk.Dec) {
	decBytes, err := rate.Marshal()
	if err != nil {
		panic(fmt.Errorf("unable to marshal rate value %v", err))
	}

	store := ctx.KVStore(k.storeKey)
	rateStore := prefix.NewStore(store, types.KeyRate)

	rateStore.Set([]byte(pairDenom), decBytes)
}

func (k Keeper) GetRate(ctx sdk.Context, pairDenom string) sdk.Dec {
	store := ctx.KVStore(k.storeKey)
	rateStore := prefix.NewStore(store, types.KeyRate)

	bz := rateStore.Get([]byte(pairDenom))
	if bz == nil {
		return sdk.NewDec(0)
	}

	var rate sdk.Dec
	err := rate.Unmarshal(bz)
	if err != nil {
		panic(fmt.Errorf("unable to unmarshal rate value %v", err))
	}
	return rate
}

func (k Keeper) SetAdmin(ctx sdk.Context, pairDenom, newAdminAddr string) {
	store := ctx.KVStore(k.storeKey)
	adminStore := prefix.NewStore(store, types.KeyAdmin)

	adminStore.Set([]byte(pairDenom), []byte(newAdminAddr))
}

func (k Keeper) GetAdmin(ctx sdk.Context, pairDenom string) string {
	store := ctx.KVStore(k.storeKey)
	adminStore := prefix.NewStore(store, types.KeyAdmin)

	bz := adminStore.Get([]byte(pairDenom))
	if bz == nil {
		return ""
	}
	return string(bz)
}

func (k Keeper) GetPaginatedCoinPairs(ctx sdk.Context, pagination *query.PageRequest) ([]types.CoinPair, *query.PageResponse, error) {
	return []types.CoinPair{}, nil, nil
}

func (k Keeper) SetCoinPairs(ctx sdk.Context, pairs []types.CoinPair) {

}

func (k Keeper) GetIBCDenom(denom, channel string) string {
	trace := transfertypes.DenomTrace{
		Path:      "transfer/" + channel,
		BaseDenom: denom,
	}
	ibcDenom := trace.IBCDenom()
	return ibcDenom
}

func (k Keeper) GetPairDenom(fromDenom, toDenom string) string {
	return strings.ToUpper(fromDenom + toDenom)
}
