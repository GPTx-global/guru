package keeper

import (
	"context"
	"fmt"

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

	err = k.SwapCoins(ctx, fromAddr, msg.ExchangeId, msg.FromDenom, msg.ToDenom, msg.Amount.Amount)
	if err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeSwap,
			sdk.NewAttribute(types.AttributeKeyAddress, msg.FromAddress),
			sdk.NewAttribute(types.AttributeKeyAmount, msg.Amount.String()),
			// sdk.NewAttribute(types.AttributeKeyRate, rate.String()),
		),
	)

	return &types.MsgSwapResponse{}, nil
}

// RegisterReserveAccount implements types.MsgServer.
func (k Keeper) RegisterAdmin(goCtx context.Context, msg *types.MsgRegisterAdmin) (*types.MsgRegisterAdminResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	moderator_address := k.GetModeratorAddress(ctx)
	if moderator_address != msg.ModeratorAddress {
		return nil, errorsmod.Wrapf(types.ErrWrongModerator, ", expected: %s, got: %s", moderator_address, msg.ModeratorAddress)
	}

	_, err := sdk.AccAddressFromBech32(msg.AdminAddress)
	if err != nil {
		return nil, err
	}

	if msg.ExchangeId.IsZero() {
		k.AddNewAdmin(ctx, types.Admin{Address: msg.AdminAddress, Exchanges: types.AdminExchanges{}})
	} else {
		k.SetAdmin(ctx, msg.AdminAddress, msg.ExchangeId)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRegisterAdmin,
			sdk.NewAttribute(types.AttributeKeyModerator, moderator_address),
			sdk.NewAttribute(types.AttributeKeyAddress, msg.AdminAddress),
		),
	)

	return &types.MsgRegisterAdminResponse{}, nil
}

// RegisterReserveAccount implements types.MsgServer.
func (k Keeper) RemoveAdmin(goCtx context.Context, msg *types.MsgRemoveAdmin) (*types.MsgRemoveAdminResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	moderator_address := k.GetModeratorAddress(ctx)
	if moderator_address != msg.ModeratorAddress {
		return nil, errorsmod.Wrapf(types.ErrWrongModerator, ", expected: %s, got: %s", moderator_address, msg.ModeratorAddress)
	}

	_, err := sdk.AccAddressFromBech32(msg.AdminAddress)
	if err != nil {
		return nil, err
	}

	k.DeleteAdmin(ctx, msg.AdminAddress)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRemoveAdmin,
			sdk.NewAttribute(types.AttributeKeyModerator, moderator_address),
			sdk.NewAttribute(types.AttributeKeyAddress, msg.AdminAddress),
		),
	)

	return &types.MsgRemoveAdminResponse{}, nil
}

// RegisterAdmin implements types.MsgServer.
func (k Keeper) RegisterExchange(goCtx context.Context, msg *types.MsgRegisterExchange) (*types.MsgRegisterExchangeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := types.ValidateExchange(msg.Exchange)
	if err != nil {
		return nil, err
	}

	// validate the ID
	nextExchangeId := k.GetNextExchangeId(ctx)
	if !msg.Exchange.Id.Equal(nextExchangeId) {
		return nil, errorsmod.Wrapf(types.ErrInvalidExchangeId, "expected: %s, got: %s", nextExchangeId, msg.Exchange.Id)
	}

	if !k.IsAdmin(ctx, msg.AdminAddress) {
		return nil, errorsmod.Wrapf(types.ErrWrongAdmin, "%s is not an admin", msg.AdminAddress)
	}

	_, err = sdk.AccAddressFromBech32(msg.AdminAddress)
	if err != nil {
		return nil, err
	}

	k.AddNewExchange(ctx, msg.Exchange)
	k.SetAdmin(ctx, msg.AdminAddress, msg.Exchange.Id)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRegisterExchange,
			sdk.NewAttribute(types.AttributeKeyAdmin, msg.AdminAddress),
			sdk.NewAttribute(types.AttributeKeyExchangeId, msg.Exchange.Id.String()),
		),
	)

	return &types.MsgRegisterExchangeResponse{}, nil
}

// UpdateRate implements types.MsgServer.
func (k Keeper) UpdateExchange(goCtx context.Context, msg *types.MsgUpdateExchange) (*types.MsgUpdateExchangeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsAdminOf(ctx, msg.AdminAddress, msg.Id) {
		return nil, errorsmod.Wrapf(types.ErrWrongAdmin, "%s is not admin of exchange %d", msg.AdminAddress, msg.Id)
	}

	k.SetExchangeAttribute(ctx, msg.Id, types.Attribute{Key: msg.Key, Value: msg.Value})

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeUpdateExchange,
			sdk.NewAttribute(types.AttributeKeyAdmin, msg.AdminAddress),
			sdk.NewAttribute(types.AttributeKeyExchangeId, msg.Id.String()),
			sdk.NewAttribute(types.AttributeKeyAttributes, fmt.Sprintf(`{"key": "%s", "value": "%s"}`, msg.Key, msg.Value)),
		),
	)

	return &types.MsgUpdateExchangeResponse{}, nil
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
