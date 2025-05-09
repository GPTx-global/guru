package keeper

import (
	"context"

	"github.com/GPTx-global/guru/x/cex/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// QueryServer implementation
var _ types.QueryServer = Keeper{}

// ModeratorAddress returns the moderator address.
func (k Keeper) ModeratorAddress(c context.Context, _ *types.QueryModeratorAddressRequest) (*types.QueryModeratorAddressResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	moderator_address := k.GetModeratorAddress(ctx)

	return &types.QueryModeratorAddressResponse{ModeratorAddress: moderator_address}, nil
}

// ReserveAccount returns the current reserve account.
func (k Keeper) ReserveAccount(c context.Context, _ *types.QueryReserveAccountRequest) (*types.QueryReserveAccountResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	address := k.GetReserveAccount(ctx)

	return &types.QueryReserveAccountResponse{Address: address}, nil
}

// Reserve returns the current reserces amount for the coin denom.
func (k Keeper) Reserve(c context.Context, req *types.QueryReserveRequest) (*types.QueryReserveResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	reserve := k.GetReserve(ctx, req.Denom)

	return &types.QueryReserveResponse{Reserve: reserve}, nil
}

// Rate returns the rate for pair_denom.
func (k Keeper) Rate(c context.Context, req *types.QueryRateRequest) (*types.QueryRateResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	rate := k.GetRate(ctx, req.PairDenom)

	return &types.QueryRateResponse{Rate: rate}, nil
}

// Admin returns the admin address for pair_denom.
func (k Keeper) Admin(c context.Context, req *types.QueryAdminRequest) (*types.QueryAdminResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	addr := k.GetAdmin(ctx, req.PairDenom)

	return &types.QueryAdminResponse{AdminAddress: addr}, nil
}
