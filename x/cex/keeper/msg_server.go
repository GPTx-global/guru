package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"github.com/GPTx-global/guru/x/cex/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MsgServer implementation
var _ types.MsgServer = &Keeper{}

// Swap implements types.MsgServer.
func (k Keeper) Swap(goCtx context.Context, msg *types.MsgSwap) (*types.MsgSwapResponse, error) {

	ctx := sdk.UnwrapSDKContext(goCtx)

	fromAddr, err := sdk.AccAddressFromBech32(msg.FromAddress)
	if err != nil {
		return nil, err
	}

	k.SwapCoins(ctx, fromAddr, msg.FromDenom, msg.FromChannel, msg.ToDenom, msg.ToChannel, msg.Amount.Amount)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeSwap,
			sdk.NewAttribute(types.AttributeKeyAddress, msg.FromAddress),
			sdk.NewAttribute(types.AttributeKeyAmount, msg.Amount.String()),
			sdk.NewAttribute(types.AttributeKeyRate, k.GetRate(ctx, k.GetPairDenom(msg.FromDenom, msg.ToDenom)).String()),
		),
	)

	return &types.MsgSwapResponse{}, nil
}

// RegisterReserveAccount implements types.MsgServer.
func (k Keeper) RegisterReserveAccount(goCtx context.Context, msg *types.MsgRegisterReserveAccount) (*types.MsgRegisterReserveAccountResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	moderator_address := k.GetModeratorAddress(ctx)
	if moderator_address != msg.ModeratorAddress {
		return nil, errorsmod.Wrapf(types.ErrWrongModerator, ", expected: %s, got: %s", moderator_address, msg.ModeratorAddress)
	}

	_, err := sdk.AccAddressFromBech32(msg.NewReserveAddress)
	if err != nil {
		return nil, err
	}

	k.SetReserveAccount(ctx, msg.NewReserveAddress)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRegisterReserveAccount,
			sdk.NewAttribute(types.AttributeKeyModerator, moderator_address),
			sdk.NewAttribute(types.AttributeKeyAddress, msg.NewReserveAddress),
		),
	)

	return &types.MsgRegisterReserveAccountResponse{}, nil
}

// RegisterAdmin implements types.MsgServer.
func (k Keeper) RegisterAdmin(goCtx context.Context, msg *types.MsgRegisterAdmin) (*types.MsgRegisterAdminResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	moderator_address := k.GetModeratorAddress(ctx)
	if moderator_address != msg.ModeratorAddress {
		return nil, errorsmod.Wrapf(types.ErrWrongModerator, ", expected: %s, got: %s", moderator_address, msg.ModeratorAddress)
	}

	_, err := sdk.AccAddressFromBech32(msg.NewAdminAddress)
	if err != nil {
		return nil, err
	}

	k.SetAdmin(ctx, msg.PairDenom, msg.NewAdminAddress)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRegisterAdmin,
			sdk.NewAttribute(types.AttributeKeyModerator, moderator_address),
			sdk.NewAttribute(types.AttributeKeyAddress, msg.NewAdminAddress),
			sdk.NewAttribute(types.AttributeKeyPairDenom, msg.PairDenom),
		),
	)

	return &types.MsgRegisterAdminResponse{}, nil
}

// UpdateRate implements types.MsgServer.
func (k Keeper) UpdateRate(goCtx context.Context, msg *types.MsgUpdateRate) (*types.MsgUpdateRateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	admin := k.GetAdmin(ctx, msg.PairDenom)
	if admin != msg.AdminAddress {
		return nil, errorsmod.Wrapf(types.ErrWrongAdmin, ", expected: %s, got: %s", admin, msg.AdminAddress)
	}

	k.SetRate(ctx, msg.PairDenom, sdk.MustNewDecFromStr(msg.NewRate))

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeUpdateRate,
			sdk.NewAttribute(types.AttributeKeyAdmin, msg.AdminAddress),
			sdk.NewAttribute(types.AttributeKeyPairDenom, msg.PairDenom),
			sdk.NewAttribute(types.AttributeKeyRate, msg.NewRate),
		),
	)

	return &types.MsgUpdateRateResponse{}, nil
}

// ChangeModerator implements types.MsgServer.
func (k Keeper) ChangeModerator(goCtx context.Context, msg *types.MsgChangeModerator) (*types.MsgChangeModeratorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	moderator_address := k.GetModeratorAddress(ctx)
	if msg.ModeratorAddress != moderator_address {
		return nil, errorsmod.Wrapf(types.ErrWrongModerator, ", expected: %s, got: %s", moderator_address, msg.ModeratorAddress)
	}

	_, err := sdk.AccAddressFromBech32(msg.NewModeratorAddress)
	if err != nil {
		return nil, err
	}

	// Update the KV store
	k.SetModeratorAddress(ctx, msg.NewModeratorAddress)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeChangeModerator,
			sdk.NewAttribute(types.AttributeKeyModerator, msg.ModeratorAddress),
			sdk.NewAttribute(types.AttributeKeyAddress, msg.NewModeratorAddress),
		),
	)

	return &types.MsgChangeModeratorResponse{}, nil
}
