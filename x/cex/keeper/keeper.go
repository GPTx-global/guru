package keeper

import (
	"encoding/binary"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/GPTx-global/guru/x/cex/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

func (k Keeper) SwapCoins(ctx sdk.Context, senderAcc sdk.AccAddress, exchangeId math.Int, fromDenom, toDenom string, fromAmount math.Int) error {
	// exchange, err := k.GetExchange(ctx, exchangeId)
	// if err != nil {
	// 	return errorsmod.Wrapf(types.ErrInvalidExchangeId, "%v", err)
	// }
	// if err := types.ValidateExchangeRequiredKeys(exchange); err != nil {
	// 	return err
	// }

	// exReserve, _ := k.GetExchangeAttribute(ctx, exchangeId, types.KeyExchangeReserveAddress)
	// exRate, _ := k.GetExchangeAttribute(ctx, exchangeId, types.KeyExchangeRate)
	// exFee, _ := k.GetExchangeAttribute(ctx, exchangeId, types.KeyExchangeFee)
	// exBaseIbc, _ := k.GetExchangeAttribute(ctx, exchangeId, types.KeyExchangeBaseIBC)
	// exBaseShort, _ := k.GetExchangeAttribute(ctx, exchangeId, types.KeyExchangeBaseShort)
	// exQuoteIbc, _ := k.GetExchangeAttribute(ctx, exchangeId, types.KeyExchangeQuoteIBC)
	// exQuoteShort, _ := k.GetExchangeAttribute(ctx, exchangeId, types.KeyExchangeQuoteShort)
	// rate := sdk.MustNewDecFromStr(exRate.Value)
	// fee := sdk.MustNewDecFromStr(exFee.Value)
	// reserveAcc := sdk.MustAccAddressFromBech32(exReserve.Value)

	// var mul bool
	// var fromIbcDenom, toIbcDenom string
	// var toAmount math.Int
	// if (fromDenom == exBaseShort.Value || fromDenom == exBaseIbc.Value) && (toDenom == exQuoteShort.Value || toDenom == exQuoteIbc.Value) {
	// 	mul = true
	// 	fromIbcDenom = exBaseIbc.Value
	// 	toIbcDenom = exQuoteIbc.Value
	// } else if (fromDenom == exQuoteShort.Value || fromDenom == exQuoteIbc.Value) && (toDenom == exBaseShort.Value || toDenom == exBaseIbc.Value) {
	// 	mul = false
	// 	fromIbcDenom = exQuoteIbc.Value
	// 	toIbcDenom = exBaseIbc.Value
	// } else {
	// 	return errorsmod.Wrapf(types.ErrInvalidExchangeCoins, "%s and %s", fromDenom, toDenom)
	// }

	// if mul {
	// 	conv := rate.MulInt(fromAmount)
	// 	toAmount = conv.Sub(conv.Mul(fee)).TruncateInt()
	// } else {
	// 	conv := sdk.NewDecFromInt(fromAmount).Quo(rate)
	// 	toAmount = conv.Sub(conv.Mul(fee)).TruncateInt()
	// }

	// senderBalance := k.bankKeeper.GetBalance(ctx, senderAcc, fromIbcDenom)

	// if senderBalance.Amount.LT(fromAmount) {
	// 	return errorsmod.Wrapf(types.ErrInsufficientBalance, "sender balance: %s is less than %s", senderBalance.String(), fromAmount.String()+fromDenom)
	// }

	// reserveBalance := k.bankKeeper.GetBalance(ctx, reserveAcc, toIbcDenom)
	// if reserveBalance.Amount.LT(toAmount) {
	// 	return errorsmod.Wrapf(types.ErrInsufficientReserve, "reserve balance: %s is less than %s", senderBalance.String(), fromAmount.String()+fromDenom)
	// }

	// err = k.bankKeeper.SendCoins(ctx, senderAcc, reserveAcc, sdk.NewCoins(sdk.NewCoin(fromIbcDenom, fromAmount)))
	// if err != nil {
	// 	return err
	// }
	// err = k.bankKeeper.SendCoins(ctx, reserveAcc, senderAcc, sdk.NewCoins(sdk.NewCoin(toIbcDenom, toAmount)))
	// if err != nil {
	// 	return err
	// }
	return nil
}

// GetModeratorAddress returns the current moderator address.
func (k Keeper) GetModeratorAddress(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyModeratorAddress)
	if bz == nil {
		return ""
	}
	return string(bz)
}

// SetModeratorAddress adds/updates the moderator address.
func (k Keeper) SetModeratorAddress(ctx sdk.Context, moderator_address string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyModeratorAddress, []byte(moderator_address))
}

// GetRatemeter returns the current ratemeter.
func (k Keeper) GetRatemeter(ctx sdk.Context) (*types.Ratemeter, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyRatemeter)
	if bz == nil {
		return nil, errorsmod.Wrapf(types.ErrNotFound, "ratemeter not found")
	}

	var ratemeter types.Ratemeter
	k.cdc.MustUnmarshal(bz, &ratemeter)

	return &ratemeter, nil
}

// SetRatemeter adds/updates the ratemeter.
func (k Keeper) SetRatemeter(ctx sdk.Context, ratemeter *types.Ratemeter) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyRatemeter, k.cdc.MustMarshal(ratemeter))
}

func (k Keeper) GetAddressRequestCount(ctx sdk.Context, address string) uint64 {
	store := ctx.KVStore(k.storeKey)
	addressRequestsStore := prefix.NewStore(store, types.GetAddressRequestCountPrefix(address))
	iterator := sdk.KVStorePrefixIterator(addressRequestsStore, nil)
	defer iterator.Close()

	ratemeter, err := k.GetRatemeter(ctx)
	if err != nil {
		return 0
	}

	now := ctx.BlockTime()
	alignedStart := now.Truncate(ratemeter.RequestPeriod)

	for ; iterator.Valid(); iterator.Next() {
		if binary.BigEndian.Uint64(iterator.Key()) == uint64(alignedStart.UnixNano()) {
			return binary.BigEndian.Uint64(iterator.Value())
		} else {
			addressRequestsStore.Delete(iterator.Key())
		}
	}

	return 0
}

func (k Keeper) SetAddressRequestCount(ctx sdk.Context, address string, count uint64) error {
	store := ctx.KVStore(k.storeKey)
	addressRequestsStore := prefix.NewStore(store, types.GetAddressRequestCountPrefix(address))

	ratemeter, err := k.GetRatemeter(ctx)
	if err != nil {
		return err
	}

	now := ctx.BlockTime()
	alignedStart := now.Truncate(ratemeter.RequestPeriod)

	idBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(idBytes, uint64(alignedStart.UnixNano()))

	addressRequestsStore.Set(idBytes, sdk.Uint64ToBigEndian(count))
	return nil
}
