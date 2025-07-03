package types

import (
	"strings"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	// DefaultRelativePacketTimeoutHeight is the default packet timeout height (in blocks) relative
	// to the current block height of the counterparty chain provided by the client state. The
	// timeout is disabled when set to 0.
	DefaultRelativePacketTimeoutHeight = "0-1000"

	// DefaultRelativePacketTimeoutTimestamp is the default packet timeout timestamp (in nanoseconds)
	// relative to the current block timestamp of the counterparty chain provided by the client
	// state. The timeout is disabled when set to 0. The default is currently set to a 10 minute
	// timeout.
	DefaultRelativePacketTimeoutTimestamp = uint64((time.Duration(10) * time.Minute).Nanoseconds())
)

// NewFungibleTokenPacketData contructs a new FungibleTokenPacketData instance
func NewFungibleTokenPacketData(
	denom string, amount math.Int,
	sender, receiver string,
	memo string,
) FungibleTokenPacketData {
	return FungibleTokenPacketData{
		Denom:    denom,
		Amount:   amount,
		Sender:   sender,
		Receiver: receiver,
		Memo:     memo,
	}
}

// ValidateBasic is used for validating the token transfer.
// NOTE: The addresses formats are not validated as the sender and recipient can have different
// formats defined by their corresponding chains that are not known to IBC.
func (ftpd FungibleTokenPacketData) ValidateBasic() error {
	if ftpd.Amount.IsNegative() || ftpd.Amount.IsZero() {
		return sdkerrors.Wrapf(ErrInvalidAmount, "amount must be strictly positive: got %d", ftpd.Amount)
	}
	if strings.TrimSpace(ftpd.Sender) == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "sender address cannot be blank")
	}
	if strings.TrimSpace(ftpd.Receiver) == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "receiver address cannot be blank")
	}
	return ValidatePrefixedDenom(ftpd.Denom)
}

// GetBytes is a helper for serialising
func (ftpd FungibleTokenPacketData) GetBytes() []byte {
	return sdk.MustSortJSON(mustProtoMarshalJSON(&ftpd))
}

// NewExchangeTokenPacketData contructs a new NewExchangeTokenPacketData instance
func NewExchangeTokenPacketData(
	packet FungibleTokenPacketData,
	cexId math.Int,
	rate *sdk.Dec,
	slippage *sdk.Dec,
) ExchangeTokenPacketData {
	return ExchangeTokenPacketData{
		Packet:   packet,
		CexId:    cexId,
		Rate:     rate,
		Slippage: slippage,
	}
}

// ValidateBasic is used for validating the token transfer.
// NOTE: The addresses formats are not validated as the sender and recipient can have different
// formats defined by their corresponding chains that are not known to IBC.
func (epd ExchangeTokenPacketData) ValidateBasic() error {
	err := epd.Packet.ValidateBasic()
	if err != nil {
		return err
	}
	if epd.CexId.IsNegative() || epd.CexId.IsZero() {
		return sdkerrors.Wrapf(ErrInvalidCexId, "cex id must be positive: got %d", epd.CexId)
	}
	if epd.Rate != nil && (epd.Rate.IsNegative() || epd.Rate.IsZero()) {
		return sdkerrors.Wrapf(ErrInvalidRate, "rate must be positive: got %s", epd.Rate)
	}
	if epd.Slippage != nil && (epd.Slippage.IsNegative() || epd.Slippage.IsZero()) {
		return sdkerrors.Wrapf(ErrInvalidSlippage, "slippage must be positive: got %s", epd.Slippage)
	}
	return nil
}

// GetBytes is a helper for serialising
func (epd ExchangeTokenPacketData) GetBytes() []byte {
	return sdk.MustSortJSON(mustProtoMarshalJSON(&epd))
}
