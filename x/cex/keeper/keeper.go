package keeper

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/GPTx-global/guru/x/cex/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
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
	exchange, err := k.GetExchange(ctx, exchangeId)
	if err != nil {
		return errorsmod.Wrapf(types.ErrInvalidExchangeId, "%v", err)
	}
	if err := types.ValidateExchangeRequiredKeys(exchange); err != nil {
		return err
	}

	exReserve, _ := k.GetExchangeAttribute(ctx, exchangeId, types.KeyExchangeReserveAddress)
	exRate, _ := k.GetExchangeAttribute(ctx, exchangeId, types.KeyExchangeRate)
	exFee, _ := k.GetExchangeAttribute(ctx, exchangeId, types.KeyExchangeFee)
	exBaseIbc, _ := k.GetExchangeAttribute(ctx, exchangeId, types.KeyExchangeBaseIBC)
	exBaseShort, _ := k.GetExchangeAttribute(ctx, exchangeId, types.KeyExchangeBaseShort)
	exQuoteIbc, _ := k.GetExchangeAttribute(ctx, exchangeId, types.KeyExchangeQuoteIBC)
	exQuoteShort, _ := k.GetExchangeAttribute(ctx, exchangeId, types.KeyExchangeQuoteShort)
	rate := sdk.MustNewDecFromStr(exRate.Value)
	fee := sdk.MustNewDecFromStr(exFee.Value)
	reserveAcc := sdk.MustAccAddressFromBech32(exReserve.Value)

	var mul bool
	var fromIbcDenom, toIbcDenom string
	var toAmount math.Int
	if (fromDenom == exBaseShort.Value || fromDenom == exBaseIbc.Value) && (toDenom == exQuoteShort.Value || toDenom == exQuoteIbc.Value) {
		mul = true
		fromIbcDenom = exBaseIbc.Value
		toIbcDenom = exQuoteIbc.Value
	} else if (fromDenom == exQuoteShort.Value || fromDenom == exQuoteIbc.Value) && (toDenom == exBaseShort.Value || toDenom == exBaseIbc.Value) {
		mul = false
		fromIbcDenom = exQuoteIbc.Value
		toIbcDenom = exBaseIbc.Value
	} else {
		return errorsmod.Wrapf(types.ErrInvalidExchangeCoins, "%s and %s", fromDenom, toDenom)
	}

	if mul {
		conv := rate.MulInt(fromAmount)
		toAmount = conv.Sub(conv.Mul(fee)).TruncateInt()
	} else {
		conv := sdk.NewDecFromInt(fromAmount).Quo(rate)
		toAmount = conv.Sub(conv.Mul(fee)).TruncateInt()
	}

	senderBalance := k.bankKeeper.GetBalance(ctx, senderAcc, fromIbcDenom)

	if senderBalance.Amount.LT(fromAmount) {
		return errorsmod.Wrapf(types.ErrInsufficientBalance, "sender balance: %s is less than %s", senderBalance.String(), fromAmount.String()+fromDenom)
	}

	reserveBalance := k.bankKeeper.GetBalance(ctx, reserveAcc, toIbcDenom)
	if reserveBalance.Amount.LT(toAmount) {
		return errorsmod.Wrapf(types.ErrInsufficientReserve, "reserve balance: %s is less than %s", senderBalance.String(), fromAmount.String()+fromDenom)
	}

	err = k.bankKeeper.SendCoins(ctx, senderAcc, reserveAcc, sdk.NewCoins(sdk.NewCoin(fromIbcDenom, fromAmount)))
	if err != nil {
		return err
	}
	err = k.bankKeeper.SendCoins(ctx, reserveAcc, senderAcc, sdk.NewCoins(sdk.NewCoin(toIbcDenom, toAmount)))
	if err != nil {
		return err
	}
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

func (k Keeper) GetExchange(ctx sdk.Context, id math.Int) (*types.Exchange, error) {
	store := ctx.KVStore(k.storeKey)
	exchangeStore := prefix.NewStore(store, types.KeyExchanges)

	idBytes, err := id.Marshal()
	if err != nil {
		panic(fmt.Errorf("unable to marshal exchange id %v", err))
	}

	bz := exchangeStore.Get(idBytes)
	if bz == nil {
		return nil, nil
	}

	var exchange types.Exchange
	k.cdc.MustUnmarshal(bz, &exchange)

	return &exchange, nil
}

func (k Keeper) GetPaginatedExchanges(ctx sdk.Context, pagination *query.PageRequest) ([]types.Exchange, *query.PageResponse, error) {
	store := ctx.KVStore(k.storeKey)
	exchangeStore := prefix.NewStore(store, types.KeyExchanges)

	exchanges := []types.Exchange{}

	pageRes, err := query.Paginate(exchangeStore, pagination, func(key, value []byte) error {
		var exchange types.Exchange
		k.cdc.MustUnmarshal(value, &exchange)

		// add the exchange to the list
		exchanges = append(exchanges, exchange)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return exchanges, pageRes, nil
}

func (k Keeper) AddNewExchange(ctx sdk.Context, exchange *types.Exchange) error {
	store := ctx.KVStore(k.storeKey)
	exchangeStore := prefix.NewStore(store, types.KeyExchanges)

	idBytes, err := exchange.Id.Marshal()
	if err != nil {
		panic(fmt.Errorf("unable to marshal exchange id %v", err))
	}

	bz := exchangeStore.Get(idBytes)
	if bz != nil {
		return errorsmod.Wrapf(types.ErrInvalidExchangeId, "exchange already exists with id %s", exchange.Id)
	}

	exchangeBytes := k.cdc.MustMarshal(exchange)
	exchangeStore.Set(idBytes, exchangeBytes)

	// update the next exchange id
	nextId := k.GetNextExchangeId(ctx)
	if nextId.LTE(exchange.Id) {
		k.setNextExchangeId(ctx, exchange.Id.Add(math.NewInt(1)))
	}

	return nil
}

func (k Keeper) SetExchange(ctx sdk.Context, exchange *types.Exchange) error {
	store := ctx.KVStore(k.storeKey)
	exchangeStore := prefix.NewStore(store, types.KeyExchanges)

	idBytes, err := exchange.Id.Marshal()
	if err != nil {
		panic(fmt.Errorf("unable to marshal exchange id %v", err))
	}

	bz := exchangeStore.Get(idBytes)
	if bz != nil {
		return errorsmod.Wrapf(types.ErrInvalidExchangeId, "exchange not found with id %s", exchange.Id)
	}

	exchangeBytes := k.cdc.MustMarshal(exchange)
	exchangeStore.Set(idBytes, exchangeBytes)
	return nil
}

func (k Keeper) GetExchangeAttribute(ctx sdk.Context, id math.Int, key string) (types.Attribute, error) {
	exchange, err := k.GetExchange(ctx, id)
	if err != nil {
		return types.Attribute{}, errorsmod.Wrapf(types.ErrInvalidExchangeId, "exchange not found with id %s", id)
	}

	for _, attr := range exchange.Attributes {
		if attr.Key == key {
			return types.Attribute{Key: key, Value: attr.Value}, nil
		}
	}

	return types.Attribute{}, errorsmod.Wrapf(types.ErrKeyNotFound, "key: %s", key)
}

func (k Keeper) SetExchangeAttribute(ctx sdk.Context, id math.Int, attribute types.Attribute) error {
	exchange, err := k.GetExchange(ctx, id)
	if err != nil {
		return errorsmod.Wrapf(types.ErrInvalidExchangeId, "exchange not found with id %s", id)
	}

	if attribute.Key == types.KeyExchangeId || attribute.Key == types.KeyExchangeBaseIBC || attribute.Key == types.KeyExchangeQuoteIBC {
		return errorsmod.Wrapf(types.ErrConstantKey, "key: %s", attribute.Key)
	}

	ok := false
	for idx, attr := range exchange.Attributes {
		if attr.Key == attribute.Key {
			exchange.Attributes[idx].Value = attribute.Value
			ok = true
			break
		}
	}
	if !ok {
		exchange.Attributes = append(exchange.Attributes, attribute)
	}

	return k.SetExchange(ctx, exchange)
}

func (k Keeper) GetNextExchangeId(ctx sdk.Context) math.Int {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyNextExchangeId)
	if bz == nil {
		return math.NewInt(1)
	}

	var id math.Int
	err := id.Unmarshal(bz)
	if err != nil {
		panic(fmt.Errorf("unable to unmarshal into exchange id %v", err))
	}

	return id
}

func (k Keeper) setNextExchangeId(ctx sdk.Context, id math.Int) {
	store := ctx.KVStore(k.storeKey)
	idBytes, err := id.Marshal()
	if err != nil {
		panic(fmt.Errorf("unable to marshal exchange id %v", err))
	}
	store.Set(types.KeyNextExchangeId, []byte(idBytes))
}
