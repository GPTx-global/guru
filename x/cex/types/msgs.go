package types

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// cex message types
const (
	TypeMsgSwap             = ModuleName + "_swap"
	TypeMsgRegisterExchange = ModuleName + "_register_exchange"
	TypeMsgUpdateExchange   = ModuleName + "_update_exchange"
	TypeMsgChangeModerator  = ModuleName + "_change_moderator_address"
)

var _ sdk.Msg = &MsgSwap{}

// NewMsgSwap - construct a msg.
func NewMsgSwap(fromAddress sdk.AccAddress, exchangeId math.Int, fromDenom, toDenom string, amount sdk.Coin) *MsgSwap {
	return &MsgSwap{
		FromAddress: fromAddress.String(),
		ExchangeId:  exchangeId,
		FromDenom:   fromDenom,
		ToDenom:     toDenom,
		Amount:      amount,
	}
}

// Route Implements Msg.
func (msg MsgSwap) Route() string { return RouterKey }

// Type Implements Msg.
func (msg MsgSwap) Type() string { return TypeMsgSwap }

// ValidateBasic Implements Msg.
func (msg MsgSwap) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.FromAddress); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, " sender address, %s", err)
	}
	if err := sdk.ValidateDenom(msg.FromDenom); err != nil {
		return errorsmod.Wrapf(ErrInvalidDenom, ", fromDenom %s", err)
	}

	if err := sdk.ValidateDenom(msg.ToDenom); err != nil {
		return errorsmod.Wrapf(ErrInvalidDenom, ", toDenom %s", err)
	}

	if !msg.Amount.IsValid() {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidCoins, " %s", msg.Amount.String())
	}

	if !msg.Amount.IsPositive() {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidCoins, " %d is not positive", msg.Amount.Amount)
	}

	return nil
}

// GetSignBytes Implements Msg.
func (msg MsgSwap) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners Implements Msg.
func (msg MsgSwap) GetSigners() []sdk.AccAddress {
	authAddress, _ := sdk.AccAddressFromBech32(msg.FromAddress)
	return []sdk.AccAddress{authAddress}
}

var _ sdk.Msg = &MsgRegisterAdmin{}

// NewMsgRegisterAdmin - construct a msg.
func NewMsgRegisterAdmin(modAddress, adminAddress sdk.AccAddress, exchangeId math.Int) *MsgRegisterAdmin {
	return &MsgRegisterAdmin{
		ModeratorAddress: modAddress.String(),
		AdminAddress:     adminAddress.String(),
		ExchangeId:       exchangeId,
	}
}

// Route Implements Msg.
func (msg MsgRegisterAdmin) Route() string { return RouterKey }

// Type Implements Msg.
func (msg MsgRegisterAdmin) Type() string { return TypeMsgRegisterExchange }

// ValidateBasic Implements Msg.
func (msg MsgRegisterAdmin) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.AdminAddress); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, " moderator address, %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.AdminAddress); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, " admin address, %s", err)
	}

	return nil
}

// GetSignBytes Implements Msg.
func (msg MsgRegisterAdmin) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners Implements Msg.
func (msg MsgRegisterAdmin) GetSigners() []sdk.AccAddress {
	modAddress, _ := sdk.AccAddressFromBech32(msg.ModeratorAddress)
	return []sdk.AccAddress{modAddress}
}

var _ sdk.Msg = &MsgRemoveAdmin{}

// NewMsgRemoveAdmin - construct a msg.
func NewMsgRemoveAdmin(modAddress, adminAddress sdk.AccAddress) *MsgRemoveAdmin {
	return &MsgRemoveAdmin{
		ModeratorAddress: modAddress.String(),
		AdminAddress:     adminAddress.String(),
	}
}

// Route Implements Msg.
func (msg MsgRemoveAdmin) Route() string { return RouterKey }

// Type Implements Msg.
func (msg MsgRemoveAdmin) Type() string { return TypeMsgRegisterExchange }

// ValidateBasic Implements Msg.
func (msg MsgRemoveAdmin) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.AdminAddress); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, " moderator address, %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.AdminAddress); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, " admin address, %s", err)
	}

	return nil
}

// GetSignBytes Implements Msg.
func (msg MsgRemoveAdmin) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners Implements Msg.
func (msg MsgRemoveAdmin) GetSigners() []sdk.AccAddress {
	modAddress, _ := sdk.AccAddressFromBech32(msg.ModeratorAddress)
	return []sdk.AccAddress{modAddress}
}

var _ sdk.Msg = &MsgRegisterExchange{}

// NewMsgRegisterExchange - construct a msg.
func NewMsgRegisterExchange(adminAddress sdk.AccAddress, exchange *Exchange) *MsgRegisterExchange {
	return &MsgRegisterExchange{
		AdminAddress: adminAddress.String(),
		Exchange:     exchange,
	}
}

// Route Implements Msg.
func (msg MsgRegisterExchange) Route() string { return RouterKey }

// Type Implements Msg.
func (msg MsgRegisterExchange) Type() string { return TypeMsgRegisterExchange }

// ValidateBasic Implements Msg.
func (msg MsgRegisterExchange) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.AdminAddress); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, " admin address, %s", err)
	}

	if err := ValidateExchange(msg.Exchange); err != nil {
		return errorsmod.Wrapf(ErrInvalidExchange, "%s", err)
	}

	return nil
}

// GetSignBytes Implements Msg.
func (msg MsgRegisterExchange) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners Implements Msg.
func (msg MsgRegisterExchange) GetSigners() []sdk.AccAddress {
	adminAddress, _ := sdk.AccAddressFromBech32(msg.AdminAddress)
	return []sdk.AccAddress{adminAddress}
}

var _ sdk.Msg = &MsgUpdateExchange{}

// NewMsgUpdateExchange - construct a msg.
func NewMsgUpdateExchange(adminAddress sdk.AccAddress, id math.Int, key, value string) *MsgUpdateExchange {
	return &MsgUpdateExchange{
		AdminAddress: adminAddress.String(),
		Id:           id,
		Key:          key,
		Value:        value,
	}
}

// Route Implements Msg.
func (msg MsgUpdateExchange) Route() string { return RouterKey }

// Type Implements Msg.
func (msg MsgUpdateExchange) Type() string { return TypeMsgUpdateExchange }

// ValidateBasic Implements Msg.
func (msg MsgUpdateExchange) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.AdminAddress); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, " admin address, %s", err)
	}

	if err := ValidateAttribute(Attribute{Key: msg.Key, Value: msg.Value}); err != nil {
		return errorsmod.Wrapf(ErrInvalidAttribute, "%s", err)
	}

	return nil
}

// GetSignBytes Implements Msg.
func (msg MsgUpdateExchange) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners Implements Msg.
func (msg MsgUpdateExchange) GetSigners() []sdk.AccAddress {
	authAddress, _ := sdk.AccAddressFromBech32(msg.AdminAddress)
	return []sdk.AccAddress{authAddress}
}

var _ sdk.Msg = &MsgChangeModerator{}

// NewMsgChangeModerator - construct a msg.
func NewMsgChangeModerator(modAddress, newModeratorAddress sdk.AccAddress) *MsgChangeModerator {
	return &MsgChangeModerator{ModeratorAddress: modAddress.String(), NewModeratorAddress: newModeratorAddress.String()}
}

// Route Implements Msg.
func (msg MsgChangeModerator) Route() string { return RouterKey }

// Type Implements Msg.
func (msg MsgChangeModerator) Type() string { return TypeMsgChangeModerator }

// ValidateBasic Implements Msg.
func (msg MsgChangeModerator) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.ModeratorAddress); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, " moderator address, %s", err)
	}

	if _, err := sdk.AccAddressFromBech32(msg.NewModeratorAddress); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, " new moderator address, %s", err)
	}

	return nil
}

// GetSignBytes Implements Msg.
func (msg MsgChangeModerator) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners Implements Msg.
func (msg MsgChangeModerator) GetSigners() []sdk.AccAddress {
	authAddress, _ := sdk.AccAddressFromBech32(msg.ModeratorAddress)
	return []sdk.AccAddress{authAddress}
}
