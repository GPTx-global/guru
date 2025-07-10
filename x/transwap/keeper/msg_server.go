package keeper

import (
	"context"

	"github.com/GPTx-global/guru/x/transwap/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
)

var _ types.MsgServer = Keeper{}

// Transfer defines a rpc handler method for MsgTeleport.
func (k Keeper) Transfer(goCtx context.Context, msg *types.MsgTransfer) (*types.MsgTransferResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.GetSendEnabled(ctx) {
		return nil, types.ErrSendDisabled
	}

	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	if !k.bankKeeper.IsSendEnabledCoin(ctx, msg.Token) {
		return nil, sdkerrors.Wrapf(types.ErrSendDisabled, "%s transfers are currently disabled", msg.Token.Denom)
	}

	if k.bankKeeper.BlockedAddr(sender) {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "%s is not allowed to send funds", sender)
	}

	sequence, err := k.sendTransfer(
		ctx, msg.SourcePort, msg.SourceChannel, msg.Token, sender, msg.Receiver, clienttypes.NewHeight(msg.TimeoutRevisionNumber, msg.TimeoutRevisionHeight), msg.TimeoutTimestamp, msg.Memo)
	if err != nil {
		return nil, err
	}

	k.Logger(ctx).Info("IBC fungible token transfer", "token", msg.Token.Denom, "amount", msg.Token.Amount.String(), "sender", msg.Sender, "receiver", msg.Receiver)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeTransfer,
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
			sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
	})

	return &types.MsgTransferResponse{Sequence: sequence}, nil
}
