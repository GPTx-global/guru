package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/GPTx-global/guru/x/cex/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
)

// QueryServer implementation
var _ types.QueryServer = Keeper{}

// ModeratorAddress returns the moderator address.
func (k Keeper) ModeratorAddress(c context.Context, _ *types.QueryModeratorAddressRequest) (*types.QueryModeratorAddressResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	moderator_address := k.GetModeratorAddress(ctx)

	return &types.QueryModeratorAddressResponse{ModeratorAddress: moderator_address}, nil
}

// Exchange returns the exchange attributes for the given key (key is optional).
func (k Keeper) Attributes(c context.Context, req *types.QueryAttributesRequest) (*types.QueryAttributesResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	id, ok := math.NewIntFromString(req.Id)
	if !ok {
		return nil, errorsmod.Wrapf(types.ErrInvalidExchange, " invalid id %s", req.Id)
	}
	attributes, err := k.GetExchangeAttribute(ctx, id, req.Key)
	if err != nil {
		return nil, err
	}

	return &types.QueryAttributesResponse{Attributes: []types.Attribute{attributes}}, nil
}

// List returns the list of all exchanges.
func (k Keeper) Exchanges(c context.Context, req *types.QueryExchangesRequest) (*types.QueryExchangesResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	exchanges, _, err := k.GetPaginatedExchanges(ctx, &query.PageRequest{Limit: query.MaxLimit})
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrNotFound, " exchanges %s", err)
	}

	return &types.QueryExchangesResponse{Exchanges: exchanges}, nil
}

func (k Keeper) IsAdmin(c context.Context, req *types.QueryIsAdminRequest) (*types.QueryIsAdminResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	ok := k.IsAdminRegistered(ctx, req.Address)

	return &types.QueryIsAdminResponse{IsAdmin: ok}, nil
}

// NextExchangeId returns the next exchange id (RegisterExchange msg should match this id).
func (k Keeper) NextExchangeId(c context.Context, _ *types.QueryNextExchangeIdRequest) (*types.QueryNextExchangeIdResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	id := k.GetNextExchangeId(ctx)

	return &types.QueryNextExchangeIdResponse{Id: id}, nil
}

func (k Keeper) Ratemeter(c context.Context, _ *types.QueryRatemeterRequest) (*types.QueryRatemeterResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	ratemeter, err := k.GetRatemeter(ctx)
	if err != nil {
		return nil, err
	}
	return &types.QueryRatemeterResponse{Ratemeter: *ratemeter}, nil
}
