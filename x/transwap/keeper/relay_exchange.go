package keeper

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/GPTx-global/guru/x/transwap/types"
	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
)

func getDenomFromAccAddress(accAddress string) string {
	// Bech32 addresses contain exactly one '1' separator.
	parts := strings.SplitN(accAddress, "1", 2)
	if len(parts) != 2 {
		return ""
	}
	return parts[0]
}

func (k Keeper) OnRecvExchangePacket(ctx sdk.Context, packet channeltypes.Packet, data types.FungibleTokenPacketData) error {
	// validate packet data upon receiving
	if err := data.ValidateBasic(); err != nil {
		return err
	}

	reqAmount, ok := math.NewIntFromString(data.Amount)
	if !ok {
		return fmt.Errorf("invalid transfer amount: %s", data.Amount)
	}

	destReceiver := data.Receiver

	cexId, ok := math.NewIntFromString(data.CexId)
	if !ok {
		return fmt.Errorf("invalid cex id: %s", data.CexId)
	}

	cexPair, err := k.cexKeeper.GetExchangePair(ctx, cexId, data.Denom, "a"+getDenomFromAccAddress(data.Receiver))
	if err != nil {
		return fmt.Errorf("failed to get exchange pair: %w", err)
	}

	count := k.cexKeeper.GetAddressRequestCount(ctx, data.Sender)
	if count >= 10 {
		return fmt.Errorf("address %s has made 10 requests in the last 10 minutes", data.Sender)
	}

	ratemeter, err := k.cexKeeper.GetRatemeter(ctx)
	if err != nil {
		return fmt.Errorf("failed to get ratemeter: %w", err)
	}

	if ratemeter.RequestCountLimit.LTE(math.NewInt(int64(count))) {
		return fmt.Errorf("address %s has made %d requests in the last %s", data.Sender, count, ratemeter.RequestPeriod.String())
	}

	if reqAmount.GT(cexPair.Limit) {
		return fmt.Errorf("amount %s is greater than the limit %s", reqAmount.String(), cexPair.Limit.String())
	}

	rate, err := sdk.NewDecFromStr(data.Rate)
	if err != nil {
		return fmt.Errorf("invalid rate: %s", data.Rate)
	}

	slippage, err := sdk.NewDecFromStr(data.Slippage)
	if err != nil {
		return fmt.Errorf("invalid slippage: %s", data.Slippage)
	}

	// Calculate acceptable rate range
	minRate := rate.Sub(slippage)
	maxRate := rate.Add(slippage)

	// Check if current rate is within acceptable range
	if cexPair.Rate.LT(minRate) || cexPair.Rate.GT(maxRate) {
		return fmt.Errorf("exchange rate %s is outside acceptable range [%s, %s]", cexPair.Rate, minRate, maxRate)
	}

	swapDec := cexPair.Rate.MulInt(reqAmount)
	swapAmount := swapDec.Sub(swapDec.Mul(cexPair.Fee)).TruncateInt()

	attributes := []sdk.Attribute{
		sdk.NewAttribute("denom", data.Denom),
		sdk.NewAttribute("amount", data.Amount),
		sdk.NewAttribute("sender", data.Sender),
		sdk.NewAttribute("receiver", data.Receiver),
		sdk.NewAttribute("memo", data.Memo),
		sdk.NewAttribute("exchange_id", data.CexId),
		sdk.NewAttribute("requested_amount", reqAmount.String()),
		sdk.NewAttribute("requested_rate", data.Rate),
		sdk.NewAttribute("slippage", data.Slippage),
		sdk.NewAttribute("actual_rate", cexPair.Rate.String()),
		sdk.NewAttribute("fee", cexPair.Fee.String()),
		sdk.NewAttribute("swap_amount", swapAmount.String()),
		sdk.NewAttribute("dest_chain_denom", cexPair.ToIbcDenom),
		sdk.NewAttribute("dest_chain_port", cexPair.PortId),
		sdk.NewAttribute("dest_chain_channel", cexPair.ChannelId),
		sdk.NewAttribute("request_count", strconv.FormatUint(count, 10)),
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"tester_event",
			attributes...,
		),
	)

	data.Receiver = cexPair.ReserveAddress
	err = k.onRecvPacket(ctx, packet, data)
	if err != nil {
		return err
	}

	destMsg := types.NewMsgTransfer(
		cexPair.PortId,
		cexPair.ChannelId,
		sdk.NewCoin(cexPair.ToIbcDenom, swapAmount),
		cexPair.ReserveAddress, destReceiver,
		0, 0, // no timeout height
		uint64(time.Now().Add(10*time.Minute).UnixNano()), // 10 minutes timeout
		"swap coins through Guru station",
	)

	_, err = k.Transfer(ctx, destMsg)
	if err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"tester_event",
			sdk.Attribute{Key: "after_recv_packet", Value: "Intercepting after receiving packet"},
		),
	)

	err = k.cexKeeper.SetAddressRequestCount(ctx, data.Sender, count+1)
	if err != nil {
		return fmt.Errorf("failed to set address request count: %w", err)
	}

	return nil
}

// refundPacketToken will unescrow and send back the tokens back to sender
// if the sending chain was the source chain. Otherwise, the sent tokens
// were burnt in the original send so new tokens are minted and sent to
// the sending address.
func (k Keeper) refundExchangePacketToken(ctx sdk.Context, packet channeltypes.Packet, data types.FungibleTokenPacketData) error {
	// NOTE: packet data type already checked in handler.go

	// parse the denomination from the full denom path
	trace := types.ParseDenomTrace(data.Denom)

	// parse the transfer amount
	transferAmount, ok := math.NewIntFromString(data.Amount)
	if !ok {
		return fmt.Errorf("invalid transfer amount: %s", data.Amount)
	}
	token := sdk.NewCoin(trace.IBCDenom(), transferAmount)

	// decode the sender address
	sender, err := sdk.AccAddressFromBech32(data.Sender)
	if err != nil {
		return err
	}

	if types.SenderChainIsSource(packet.GetSourcePort(), packet.GetSourceChannel(), data.Denom) {
		// unescrow tokens back to sender
		escrowAddress := types.GetEscrowAddress(packet.GetSourcePort(), packet.GetSourceChannel())
		if err := k.bankKeeper.SendCoins(ctx, escrowAddress, sender, sdk.NewCoins(token)); err != nil {
			// NOTE: this error is only expected to occur given an unexpected bug or a malicious
			// counterparty module. The bug may occur in bank or any part of the code that allows
			// the escrow address to be drained. A malicious counterparty module could drain the
			// escrow address by allowing more tokens to be sent back then were escrowed.
			return sdkerrors.Wrap(err, "unable to unescrow tokens, this may be caused by a malicious counterparty module or a bug: please open an issue on counterparty module")
		}

		return nil
	}

	// mint vouchers back to sender
	if err := k.bankKeeper.MintCoins(
		ctx, types.ModuleName, sdk.NewCoins(token),
	); err != nil {
		return err
	}

	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, sender, sdk.NewCoins(token)); err != nil {
		panic(fmt.Sprintf("unable to send coins from module to account despite previously minting coins to module account: %v", err))
	}

	return nil
}

// OnAcknowledgementExchangePacket responds to the the success or failure of a packet
// acknowledgement written on the receiving chain. If the acknowledgement
// was a success then nothing occurs. If the acknowledgement failed, then
// the sender is refunded their tokens using the refundPacketToken function.
func (k Keeper) OnAcknowledgementExchangePacket(ctx sdk.Context, packet channeltypes.Packet, data types.FungibleTokenPacketData, ack channeltypes.Acknowledgement) error {
	switch ack.Response.(type) {
	case *channeltypes.Acknowledgement_Error:
		return k.refundExchangePacketToken(ctx, packet, data)
	default:
		// the acknowledgement succeeded on the receiving chain so nothing
		// needs to be executed and no error needs to be returned
		return nil
	}
}

// OnTimeoutExchangePacket refunds the sender since the original packet sent was
// never received and has been timed out.
func (k Keeper) OnTimeoutExchangePacket(ctx sdk.Context, packet channeltypes.Packet, data types.FungibleTokenPacketData) error {
	return k.refundExchangePacketToken(ctx, packet, data)
}
