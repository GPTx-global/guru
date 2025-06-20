package keeper

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/ibc-go/v6/modules/apps/exchange/types"

	clienttypes "github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
)

// OnRecvPacket processes a cross chain fungible token transfer. If the
// sender chain is the source of minted tokens then vouchers will be minted
// and sent to the receiving address. Otherwise if the sender chain is sending
// back tokens this chain originally transferred to it, the tokens are
// unescrowed and sent to the receiving address.
func (k Keeper) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet, data types.FungibleTokenPacketData) error {

	// just receive if the receiver is on guru chain
	_, err := sdk.AccAddressFromBech32(data.Receiver)
	if err == nil {
		return k.OnRecvTransferPacket(ctx, packet, data)
	}

	return k.OnRecvExchangePacket(sdk.WrapSDKContext(ctx), packet, data)
}

func (k Keeper) OnRecvTransferPacket(ctx sdk.Context, packet channeltypes.Packet, data types.FungibleTokenPacketData) error {
	return k.Keeper.OnRecvPacket(ctx, packet, data)
}

func (k Keeper) OnRecvExchangePacket(goCtx context.Context, packet channeltypes.Packet, data types.FungibleTokenPacketData) error {

	ctx := sdk.UnwrapSDKContext(goCtx)

	// use a zero gas config to avoid extra costs for the relayers
	kvGasCfg := ctx.KVGasConfig()
	transientKVGasCfg := ctx.TransientKVGasConfig()

	// use a zero gas config to avoid extra costs for the relayers
	ctx = ctx.
		WithKVGasConfig(storetypes.GasConfig{}).
		WithTransientKVGasConfig(storetypes.GasConfig{})

	defer func() {
		// return the KV gas config to initial values
		ctx = ctx.
			WithKVGasConfig(kvGasCfg).
			WithTransientKVGasConfig(transientKVGasCfg)
	}()

	exchangeId, ok := math.NewIntFromString(data.Args[0])
	if !ok {
		return fmt.Errorf("invalid exchange id: %s", data.Args[0])
	}

	reqRate, err := sdk.NewDecFromStr(data.Args[1])
	if err != nil {
		return fmt.Errorf("invalid requested rate: %s", data.Args[1])
	}

	reqSlippage, err := sdk.NewDecFromStr(data.Args[2])
	if err != nil {
		return fmt.Errorf("invalid requested slippage: %s", data.Args[2])
	}

	reqAmount, ok := math.NewIntFromString(data.Amount)
	if !ok {
		return fmt.Errorf("invalid requested amount: %s", data.Amount)
	}

	destReceiver := data.Receiver

	cexPair, err := k.cexKeeper.GetExchangePair(ctx, exchangeId, data.Denom, "a"+getDenomFromAccAddress(data.Receiver))
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
		return fmt.Errorf("amount %s is greater than the limit %s", data.Amount, cexPair.Limit.String())
	}

	// Calculate acceptable rate range
	minRate := reqRate.Sub(reqSlippage)
	maxRate := reqRate.Add(reqSlippage)

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
		sdk.NewAttribute("exchange_id", exchangeId.String()),
		sdk.NewAttribute("requested_amount", reqAmount.String()),
		sdk.NewAttribute("requested_rate", reqRate.String()),
		sdk.NewAttribute("slippage", reqSlippage.String()),
		sdk.NewAttribute("actual_rate", cexPair.Rate.String()),
		sdk.NewAttribute("fee", cexPair.Fee.String()),
		sdk.NewAttribute("swap_amount", swapAmount.String()),
		sdk.NewAttribute("dest_chain_denom", cexPair.ToIbcDenom),
		sdk.NewAttribute("dest_chain_port", cexPair.PortId),
		sdk.NewAttribute("dest_chain_channel", cexPair.ChannelId),
		sdk.NewAttribute("request_count", strconv.FormatUint(count, 10)),
	}
	for i, arg := range data.Args {
		attributes = append(attributes, sdk.NewAttribute(fmt.Sprintf("arg[%d]", i), arg))
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"tester_event",
			attributes...,
		),
	)

	data.Receiver = cexPair.ReserveAddress
	err = k.Keeper.OnRecvPacket(ctx, packet, data)
	if err != nil {
		return err
	}

	destMsg := types.NewMsgTeleport(cexPair.PortId, cexPair.ChannelId, sdk.NewCoin(cexPair.ToIbcDenom, swapAmount), cexPair.ReserveAddress, destReceiver, clienttypes.ZeroHeight(), uint64(time.Now().Add(10*time.Minute).UnixNano()), // clienttypes.NewHeight(0, 1000), 600000000000,
		"swap coins through Guru station", []string{},
	)

	_, err = k.Keeper.Teleport(sdk.WrapSDKContext(ctx), destMsg)
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

func getDenomFromAccAddress(accAddress string) string {
	// Bech32 addresses contain exactly one '1' separator.
	parts := strings.SplitN(accAddress, "1", 2)
	if len(parts) != 2 {
		return ""
	}
	return parts[0]
}
