package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
)

// OnRecvPacket processes a cross chain fungible token transfer. If the
// sender chain is the source of minted tokens then vouchers will be minted
// and sent to the receiving address. Otherwise if the sender chain is sending
// back tokens this chain originally transferred to it, the tokens are
// unescrowed and sent to the receiving address.
func (k Keeper) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet, data types.FungibleTokenPacketData) error {

	send := sdk.MustAccAddressFromBech32("guru1p3r3wvvm7hl8udelwhzrkc23jsvtt5wur5w9c9")
	rcv := sdk.MustAccAddressFromBech32(data.Receiver)
	amt, _ := sdk.NewIntFromString(data.Amount)

	k.bankKeeper.SendCoins(ctx, send, rcv, sdk.NewCoins(sdk.NewCoin("aguru", amt)))

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"tester_event",
			sdk.Attribute{Key: "test_key", Value: "test_value"},
		),
	)

	return k.Keeper.OnRecvPacket(ctx, packet, data)

	// return fmt.Errorf("we cannot execute because it is test 2")

}
