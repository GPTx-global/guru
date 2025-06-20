package keeper

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/GPTx-global/guru/x/cex/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
)

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

func (k Keeper) AddExchange(ctx sdk.Context, exchange *types.Exchange) error {
	store := ctx.KVStore(k.storeKey)
	exchangeStore := prefix.NewStore(store, types.KeyExchanges)

	idBytes, err := exchange.Id.Marshal()
	if err != nil {
		panic(fmt.Errorf("unable to marshal exchange id %v", err))
	}

	bz := exchangeStore.Get(idBytes)
	if bz != nil {
		return errorsmod.Wrapf(types.ErrInvalidExchange, "exchange already exists with id %s", exchange.Id)
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
	if bz == nil {
		return errorsmod.Wrapf(types.ErrInvalidExchange, "exchange not found with id %s", exchange.Id)
	}

	exchangeBytes := k.cdc.MustMarshal(exchange)
	exchangeStore.Set(idBytes, exchangeBytes)
	return nil
}

func (k Keeper) GetExchangeAttribute(ctx sdk.Context, id math.Int, key string) (types.Attribute, error) {
	exchange, err := k.GetExchange(ctx, id)
	if err != nil {
		return types.Attribute{}, errorsmod.Wrapf(types.ErrInvalidExchange, "exchange not found with id %s", id)
	}

	for _, attr := range exchange.Attributes {
		if attr.Key == key {
			return types.Attribute{Key: key, Value: attr.Value}, nil
		}
	}

	return types.Attribute{}, errorsmod.Wrapf(types.ErrNotFound, "key: %s", key)
}

func (k Keeper) SetExchangeAttribute(ctx sdk.Context, id math.Int, attribute types.Attribute) error {
	exchange, err := k.GetExchange(ctx, id)
	if err != nil {
		return errorsmod.Wrapf(types.ErrInvalidExchange, "exchange not found with id %s", id)
	}

	if attribute.Key == types.KeyExchangeId || attribute.Key == types.KeyExchangeCoinAIBCDenom || attribute.Key == types.KeyExchangeCoinBIBCDenom {
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

func (k Keeper) GetExchangePair(ctx sdk.Context, id math.Int, fromShortDenom, toShortDenom string) (*types.Pair, error) {
	exchange, err := k.GetExchange(ctx, id)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidExchange, "exchange not found with id %s", id)
	}
	attributes := types.AttributesToMap(exchange.Attributes)

	if attributes[types.KeyExchangeCoinAShort] == fromShortDenom && attributes[types.KeyExchangeCoinBShort] == toShortDenom {
		limit, ok := math.NewIntFromString(attributes[types.KeyExchangeAToBLimit])
		if !ok {
			return nil, errorsmod.Wrapf(types.ErrInvalidExchange, "invalid limit: %s", attributes[types.KeyExchangeAToBLimit])
		}
		return &types.Pair{
			ReserveAddress: attributes[types.KeyExchangeReserveAddress],
			FromIbcDenom:   attributes[types.KeyExchangeCoinAIBCDenom],
			ToIbcDenom:     attributes[types.KeyExchangeCoinBIBCDenom],
			PortId:         attributes[types.KeyExchangeCoinBPort],
			ChannelId:      attributes[types.KeyExchangeCoinBChannel],
			Rate:           sdk.MustNewDecFromStr(attributes[types.KeyExchangeAtoBRate]),
			Fee:            sdk.MustNewDecFromStr(attributes[types.KeyExchangeFee]),
			Limit:          limit,
		}, nil
	} else if attributes[types.KeyExchangeCoinAShort] == toShortDenom && attributes[types.KeyExchangeCoinBShort] == fromShortDenom {
		limit, ok := math.NewIntFromString(attributes[types.KeyExchangeBtoALimit])
		if !ok {
			return nil, errorsmod.Wrapf(types.ErrInvalidExchange, "invalid limit: %s", attributes[types.KeyExchangeBtoALimit])
		}
		return &types.Pair{
			ReserveAddress: attributes[types.KeyExchangeReserveAddress],
			FromIbcDenom:   attributes[types.KeyExchangeCoinBIBCDenom],
			ToIbcDenom:     attributes[types.KeyExchangeCoinAIBCDenom],
			PortId:         attributes[types.KeyExchangeCoinAPort],
			ChannelId:      attributes[types.KeyExchangeCoinAChannel],
			Rate:           sdk.MustNewDecFromStr(attributes[types.KeyExchangeBtoARate]),
			Fee:            sdk.MustNewDecFromStr(attributes[types.KeyExchangeFee]),
			Limit:          limit,
		}, nil
	}

	return nil, errorsmod.Wrapf(types.ErrNotFound, "pair not found with fromShortDenom: %s and toShortDenom: %s", fromShortDenom, toShortDenom)
}
