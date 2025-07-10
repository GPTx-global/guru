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
	pType string,
	denom, amount string,
	sender, receiver string,
	memo string,
	cexId, rate, slippage string,
) FungibleTokenPacketData {
	return FungibleTokenPacketData{
		Type:     pType,
		Denom:    denom,
		Amount:   amount,
		Sender:   sender,
		Receiver: receiver,
		Memo:     memo,
		CexId:    cexId,
		Rate:     rate,
		Slippage: slippage,
	}
}

// ValidateBasic is used for validating the token transfer.
// NOTE: The addresses formats are not validated as the sender and recipient can have different
// formats defined by their corresponding chains that are not known to IBC.
func (ftpd FungibleTokenPacketData) ValidateBasic() error {
	if _, ok := math.NewIntFromString(ftpd.Amount); !ok {
		return sdkerrors.Wrapf(ErrInvalidAmount, "amount must be a valid integer: got %s", ftpd.Amount)
	}
	if strings.TrimSpace(ftpd.Sender) == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "sender address cannot be blank")
	}
	if strings.TrimSpace(ftpd.Receiver) == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "receiver address cannot be blank")
	}
	if err := ValidatePrefixedDenom(ftpd.Denom); err != nil {
		return err
	}
	if strings.TrimSpace(ftpd.Type) == PacketTypeExchange {
		if _, ok := math.NewIntFromString(ftpd.CexId); !ok {
			return sdkerrors.Wrapf(ErrInvalidCexId, "cex id must be a valid integer: got %s", ftpd.CexId)
		}
		if strings.TrimSpace(ftpd.Rate) != "" {
			if _, err := sdk.NewDecFromStr(ftpd.Rate); err != nil {
				return sdkerrors.Wrapf(ErrInvalidRate, "%s", err)
			}
		}
		if strings.TrimSpace(ftpd.Slippage) != "" {
			if _, err := sdk.NewDecFromStr(ftpd.Slippage); err != nil {
				return sdkerrors.Wrapf(ErrInvalidSlippage, "%s", err)
			}
		}
	}

	return nil
}

// GetBytes is a helper for serialising
func (ftpd FungibleTokenPacketData) GetBytes() []byte {
	return sdk.MustSortJSON(mustProtoMarshalJSON(&ftpd))
}
