package keeper

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

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

func (k Keeper) OnRecvExchangePacket(ctx sdk.Context, packet channeltypes.Packet, data types.ExchangeTokenPacketData) error {
	// validate packet data upon receiving
	if err := data.ValidateBasic(); err != nil {
		return err
	}

	reqAmount := data.Packet.Amount

	destReceiver := data.Packet.Receiver

	cexPair, err := k.cexKeeper.GetExchangePair(ctx, data.CexId, data.Packet.Denom, "a"+getDenomFromAccAddress(data.Packet.Receiver))
	if err != nil {
		return fmt.Errorf("failed to get exchange pair: %w", err)
	}

	count := k.cexKeeper.GetAddressRequestCount(ctx, data.Packet.Sender)
	if count >= 10 {
		return fmt.Errorf("address %s has made 10 requests in the last 10 minutes", data.Packet.Sender)
	}

	ratemeter, err := k.cexKeeper.GetRatemeter(ctx)
	if err != nil {
		return fmt.Errorf("failed to get ratemeter: %w", err)
	}

	if ratemeter.RequestCountLimit.LTE(math.NewInt(int64(count))) {
		return fmt.Errorf("address %s has made %d requests in the last %s", data.Packet.Sender, count, ratemeter.RequestPeriod.String())
	}

	if reqAmount.GT(cexPair.Limit) {
		return fmt.Errorf("amount %s is greater than the limit %s", reqAmount.String(), cexPair.Limit.String())
	}

	// Calculate acceptable rate range
	minRate := data.Rate.Sub(*data.Slippage)
	maxRate := data.Rate.Add(*data.Slippage)

	// Check if current rate is within acceptable range
	if cexPair.Rate.LT(minRate) || cexPair.Rate.GT(maxRate) {
		return fmt.Errorf("exchange rate %s is outside acceptable range [%s, %s]", cexPair.Rate, minRate, maxRate)
	}

	swapDec := cexPair.Rate.MulInt(reqAmount)
	swapAmount := swapDec.Sub(swapDec.Mul(cexPair.Fee)).TruncateInt()

	attributes := []sdk.Attribute{
		sdk.NewAttribute("denom", data.Packet.Denom),
		sdk.NewAttribute("amount", data.Packet.Amount.String()),
		sdk.NewAttribute("sender", data.Packet.Sender),
		sdk.NewAttribute("receiver", data.Packet.Receiver),
		sdk.NewAttribute("memo", data.Packet.Memo),
		sdk.NewAttribute("exchange_id", data.CexId.String()),
		sdk.NewAttribute("requested_amount", reqAmount.String()),
		sdk.NewAttribute("requested_rate", data.Rate.String()),
		sdk.NewAttribute("slippage", data.Slippage.String()),
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

	data.Packet.Receiver = cexPair.ReserveAddress
	err = k.onRecvPacket(ctx, packet, data.Packet)
	if err != nil {
		return err
	}

	destMsg := types.NewMsgTransfer(
		cexPair.PortId,
		cexPair.ChannelId,
		sdk.NewCoin(cexPair.ToIbcDenom, swapAmount),
		cexPair.ReserveAddress, destReceiver,
		0, // no timeout height
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

	err = k.cexKeeper.SetAddressRequestCount(ctx, data.Packet.Sender, count+1)
	if err != nil {
		return fmt.Errorf("failed to set address request count: %w", err)
	}

	return nil
}

// OnAcknowledgementExchangePacket responds to the the success or failure of a packet
// acknowledgement written on the receiving chain. If the acknowledgement
// was a success then nothing occurs. If the acknowledgement failed, then
// the sender is refunded their tokens using the refundPacketToken function.
func (k Keeper) OnAcknowledgementExchangePacket(ctx sdk.Context, packet channeltypes.Packet, data types.ExchangeTokenPacketData, ack channeltypes.Acknowledgement) error {
	switch ack.Response.(type) {
	case *channeltypes.Acknowledgement_Error:
		return k.refundPacketToken(ctx, packet, data.Packet)
	default:
		// the acknowledgement succeeded on the receiving chain so nothing
		// needs to be executed and no error needs to be returned
		return nil
	}
}

// OnTimeoutExchangePacket refunds the sender since the original packet sent was
// never received and has been timed out.
func (k Keeper) OnTimeoutExchangePacket(ctx sdk.Context, packet channeltypes.Packet, data types.ExchangeTokenPacketData) error {
	return k.refundPacketToken(ctx, packet, data.Packet)
}
