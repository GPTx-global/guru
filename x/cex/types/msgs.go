package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// cex message types
const (
	TypeMsgSwap                   = ModuleName + "_swap"
	TypeMsgRegisterReserveAccount = ModuleName + "_register_reserve_account"
	TypeMsgRegisterAdmin          = ModuleName + "_register_admin"
	TypeMsgUpdateRate             = ModuleName + "_update_rate"
	TypeMsgChangeModerator        = ModuleName + "_change_moderator_address"
)

var _ sdk.Msg = &MsgSwap{}

// NewMsgSwap - construct a msg to swap coins.
func NewMsgSwap(fromAddress sdk.AccAddress, fromDenom, fromChannel, toDenom, toChannel string, amount sdk.Coin) *MsgSwap {
	return &MsgSwap{
		FromAddress: fromAddress.String(),
		FromDenom:   fromDenom,
		FromChannel: fromChannel,
		ToDenom:     toDenom,
		ToChannel:   toChannel,
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
		return errorsmod.Wrapf(ErrInvalidPairDenom, ", fromDenom %s", err)
	}

	if err := sdk.ValidateDenom(msg.ToDenom); err != nil {
		return errorsmod.Wrapf(ErrInvalidPairDenom, ", toDenom %s", err)
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

var _ sdk.Msg = &MsgRegisterReserveAccount{}

// NewMsgRegisterReserveAccount - construct a msg to register new reserve account.
func NewMsgRegisterReserveAccount(modAddr, newReserveAddress sdk.AccAddress) *MsgRegisterReserveAccount {
	return &MsgRegisterReserveAccount{ModeratorAddress: modAddr.String(), NewReserveAddress: newReserveAddress.String()}
}

// Route Implements Msg.
func (msg MsgRegisterReserveAccount) Route() string { return RouterKey }

// Type Implements Msg.
func (msg MsgRegisterReserveAccount) Type() string { return TypeMsgRegisterReserveAccount }

// ValidateBasic Implements Msg.
func (msg MsgRegisterReserveAccount) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.ModeratorAddress); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, " moderator address, %s", err)
	}

	if _, err := sdk.AccAddressFromBech32(msg.NewReserveAddress); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, " new reserve address, %s", err)
	}

	return nil
}

// GetSignBytes Implements Msg.
func (msg MsgRegisterReserveAccount) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners Implements Msg.
func (msg MsgRegisterReserveAccount) GetSigners() []sdk.AccAddress {
	authAddress, _ := sdk.AccAddressFromBech32(msg.ModeratorAddress)
	return []sdk.AccAddress{authAddress}
}

var _ sdk.Msg = &MsgRegisterAdmin{}

// NewMsgRegisterAdmin - construct arbitrary register admin msg.
func NewMsgRegisterAdmin(modAddress, newAdminAddress sdk.AccAddress, pairDenom string) *MsgRegisterAdmin {
	return &MsgRegisterAdmin{
		ModeratorAddress: modAddress.String(),
		PairDenom:        pairDenom,
		NewAdminAddress:  newAdminAddress.String(),
	}
}

// Route Implements Msg.
func (msg MsgRegisterAdmin) Route() string { return RouterKey }

// Type Implements Msg.
func (msg MsgRegisterAdmin) Type() string { return TypeMsgRegisterAdmin }

// ValidateBasic Implements Msg.
func (msg MsgRegisterAdmin) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.ModeratorAddress); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, " moderator address, %s", err)
	}

	if _, err := sdk.AccAddressFromBech32(msg.NewAdminAddress); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, " new admin address, %s", err)
	}

	if msg.PairDenom == "" {
		return errorsmod.Wrapf(ErrInvalidPairDenom, " empty denom")
	}

	return nil
}

// GetSignBytes Implements Msg.
func (msg MsgRegisterAdmin) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners Implements Msg.
func (msg MsgRegisterAdmin) GetSigners() []sdk.AccAddress {
	authAddress, _ := sdk.AccAddressFromBech32(msg.ModeratorAddress)
	return []sdk.AccAddress{authAddress}
}

var _ sdk.Msg = &MsgUpdateRate{}

// NewMsgUpdateRate - construct arbitrary update rate msg.
func NewMsgUpdateRate(adminAddress sdk.AccAddress, pairDenom string, newRate sdk.Dec) *MsgUpdateRate {
	return &MsgUpdateRate{
		AdminAddress: adminAddress.String(),
		NewRate:      newRate.String(),
		PairDenom:    pairDenom,
	}
}

// Route Implements Msg.
func (msg MsgUpdateRate) Route() string { return RouterKey }

// Type Implements Msg.
func (msg MsgUpdateRate) Type() string { return TypeMsgRegisterAdmin }

// ValidateBasic Implements Msg.
func (msg MsgUpdateRate) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.AdminAddress); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, " admin address, %s", err)
	}

	if msg.PairDenom == "" {
		return errorsmod.Wrapf(ErrInvalidPairDenom, " empty denom")
	}

	return nil
}

// GetSignBytes Implements Msg.
func (msg MsgUpdateRate) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners Implements Msg.
func (msg MsgUpdateRate) GetSigners() []sdk.AccAddress {
	authAddress, _ := sdk.AccAddressFromBech32(msg.AdminAddress)
	return []sdk.AccAddress{authAddress}
}

var _ sdk.Msg = &MsgChangeModerator{}

// NewMsgChangeModerator - construct arbitrary change moderator address msg.
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
