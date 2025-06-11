package keeper

import (
	"fmt"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cextypes "github.com/GPTx-global/guru/x/cex/types"
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

	return k.OnRecvExchangePacket(ctx, packet, data)
}

func (k Keeper) OnRecvTransferPacket(ctx sdk.Context, packet channeltypes.Packet, data types.FungibleTokenPacketData) error {
	return k.Keeper.OnRecvPacket(ctx, packet, data)
}

func (k Keeper) OnRecvExchangePacket(ctx sdk.Context, packet channeltypes.Packet, data types.FungibleTokenPacketData) error {
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

	exchange, err := k.cexKeeper.GetExchange(ctx, exchangeId)
	if err != nil {
		return fmt.Errorf("failed to get exchange: %w", err)
	}

	destReceiver := data.Receiver

	err = cextypes.ValidateExchangeRequiredKeys(exchange)
	if err != nil {
		return err
	}

	attributesMap := cextypes.AttributesToMap(exchange.Attributes)

	var swapAmount math.Int
	var destChainDenom string
	var destChainPort string
	var destChainChannel string
	var rate sdk.Dec

	fee, err := sdk.NewDecFromStr(attributesMap[cextypes.KeyExchangeFee])
	if err != nil {
		return fmt.Errorf("invalid fee: %s", attributesMap[cextypes.KeyExchangeFee])
	}

	if attributesMap[cextypes.KeyExchangeCoinAShort] == data.Denom {
		// For quote coin, divide by rate and subtract fee
		rate, err = sdk.NewDecFromStr(attributesMap[cextypes.KeyExchangeAtoBRate])
		if err != nil {
			return fmt.Errorf("invalid rate: %s", attributesMap[cextypes.KeyExchangeAtoBRate])
		}
		destChainDenom = attributesMap[cextypes.KeyExchangeCoinBIBCDenom]
		destChainPort = attributesMap[cextypes.KeyExchangeCoinBPort]
		destChainChannel = attributesMap[cextypes.KeyExchangeCoinBChannel]

	} else if attributesMap[cextypes.KeyExchangeCoinBShort] == data.Denom {
		// For base coin, multiply by rate and subtract fee
		rate, err = sdk.NewDecFromStr(attributesMap[cextypes.KeyExchangeBtoARate])
		if err != nil {
			return fmt.Errorf("invalid rate: %s", attributesMap[cextypes.KeyExchangeBtoARate])
		}
		destChainDenom = attributesMap[cextypes.KeyExchangeCoinAIBCDenom]
		destChainPort = attributesMap[cextypes.KeyExchangeCoinAPort]
		destChainChannel = attributesMap[cextypes.KeyExchangeCoinAChannel]
	} else {
		return fmt.Errorf("coin a short denom %s or coin b short denom %s does not match expected %s", attributesMap[cextypes.KeyExchangeCoinAShort], attributesMap[cextypes.KeyExchangeCoinBShort], data.Denom)
	}

	// Calculate acceptable rate range
	minRate := reqRate.Sub(reqSlippage)
	maxRate := reqRate.Add(reqSlippage)

	// Check if current rate is within acceptable range
	if rate.LT(minRate) || rate.GT(maxRate) {
		return fmt.Errorf("exchange rate %s is outside acceptable range [%s, %s]", rate, minRate, maxRate)
	}

	swapDec := rate.MulInt(reqAmount)
	swapAmount = swapDec.Sub(swapDec.Mul(fee)).TruncateInt()

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
		sdk.NewAttribute("actual_rate", rate.String()),
		sdk.NewAttribute("fee", fee.String()),
		sdk.NewAttribute("swap_amount", swapAmount.String()),
		sdk.NewAttribute("dest_chain_denom", destChainDenom),
		sdk.NewAttribute("dest_chain_port", destChainPort),
		sdk.NewAttribute("dest_chain_channel", destChainChannel),
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

	data.Receiver = attributesMap[cextypes.KeyExchangeReserveAddress]
	err = k.Keeper.OnRecvPacket(ctx, packet, data)
	if err != nil {
		return err
	}

	destMsg := types.NewMsgTeleport(destChainPort, destChainChannel, sdk.NewCoin(destChainDenom, swapAmount), attributesMap[cextypes.KeyExchangeReserveAddress], destReceiver, clienttypes.ZeroHeight(), uint64(time.Now().Add(10*time.Minute).UnixNano()), // clienttypes.NewHeight(0, 1000), 600000000000,
		"swap coins through Guru station", []string{},
	)

	_, err = k.Keeper.Teleport(ctx, destMsg)
	if err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"tester_event",
			sdk.Attribute{Key: "after_recv_packet", Value: "Intercepting after receiving packet"},
		),
	)

	return nil
}
