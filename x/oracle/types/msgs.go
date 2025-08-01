package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// NewMsgRegisterOracleRequestDoc creates a new MsgRegisterOracleRequestDoc instance
func NewMsgRegisterOracleRequestDoc(
	moderatorAddress string,
	requestDoc OracleRequestDoc,
) *MsgRegisterOracleRequestDoc {
	return &MsgRegisterOracleRequestDoc{
		ModeratorAddress: moderatorAddress,
		RequestDoc:       requestDoc,
	}
}

// Route implements the sdk.Msg interface
func (msg MsgRegisterOracleRequestDoc) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface
func (msg MsgRegisterOracleRequestDoc) Type() string {
	return "register_oracle_request_doc"
}

// GetSigners implements the sdk.Msg interface
func (msg MsgRegisterOracleRequestDoc) GetSigners() []sdk.AccAddress {
	moderatorAddress, err := sdk.AccAddressFromBech32(msg.ModeratorAddress)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{moderatorAddress}
}

// GetSignBytes implements the sdk.Msg interface
func (msg MsgRegisterOracleRequestDoc) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface
func (msg MsgRegisterOracleRequestDoc) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.ModeratorAddress); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid from address(Moderator) (%s)", err)
	}
	if err := msg.RequestDoc.Validate(); err != nil {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	}
	return nil
}

// NewMsgUpdateOracleRequestDoc creates a new MsgUpdateOracleRequestDoc instance
func NewMsgUpdateOracleRequestDoc(
	moderatorAddress string,
	requestDoc OracleRequestDoc,
	reason string,
) *MsgUpdateOracleRequestDoc {
	return &MsgUpdateOracleRequestDoc{
		ModeratorAddress: moderatorAddress,
		RequestDoc:       requestDoc,
		Reason:           reason,
	}
}

// Route implements the sdk.Msg interface
func (msg MsgUpdateOracleRequestDoc) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface
func (msg MsgUpdateOracleRequestDoc) Type() string {
	return "update_oracle_request_doc"
}

// GetSigners implements the sdk.Msg interface
func (msg MsgUpdateOracleRequestDoc) GetSigners() []sdk.AccAddress {
	moderatorAddress, err := sdk.AccAddressFromBech32(msg.ModeratorAddress)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{moderatorAddress}
}

// GetSignBytes implements the sdk.Msg interface
func (msg MsgUpdateOracleRequestDoc) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface
func (msg MsgUpdateOracleRequestDoc) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.ModeratorAddress); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid from address(Moderator) (%s)", err)
	}
	// if err := msg.RequestDoc.Validate(); err != nil {
	// 	return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	// }
	return nil
}

// NewMsgSubmitOracleData creates a new MsgSubmitOracleData instance
func NewMsgSubmitOracleData(
	requestId uint64,
	nonce uint64,
	rawData string,
	provider string,
	signature string,
	authorityAddress string,
) *MsgSubmitOracleData {
	return &MsgSubmitOracleData{
		AuthorityAddress: authorityAddress,
		DataSet: &SubmitDataSet{
			RequestId: requestId,
			Nonce:     nonce,
			RawData:   rawData,
			Provider:  provider,
			Signature: signature,
		},
	}
}

// Route implements the sdk.Msg interface
func (msg MsgSubmitOracleData) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface
func (msg MsgSubmitOracleData) Type() string {
	return "submit_oracle_data"
}

// GetSigners implements the sdk.Msg interface
func (msg MsgSubmitOracleData) GetSigners() []sdk.AccAddress {
	authorityAddress, err := sdk.AccAddressFromBech32(msg.AuthorityAddress)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{authorityAddress}
}

// GetSignBytes implements the sdk.Msg interface
func (msg MsgSubmitOracleData) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface
func (msg MsgSubmitOracleData) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.DataSet.Provider); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid provider address (%s)", err)
	}
	if msg.DataSet.RequestId == 0 {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "request ID cannot be empty")
	}
	if msg.DataSet.RawData == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "raw data cannot be empty")
	}
	if msg.DataSet.Signature == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "signature cannot be empty")
	}
	return nil
}

// NewMsgUpdateModeratorAddress creates a new MsgUpdateModeratorAddress instance
func NewMsgUpdateModeratorAddress(moderatorAddress string, newModeratorAddress string) *MsgUpdateModeratorAddress {
	return &MsgUpdateModeratorAddress{
		ModeratorAddress:    moderatorAddress,
		NewModeratorAddress: newModeratorAddress,
	}
}

// Route implements the sdk.Msg interface
func (msg MsgUpdateModeratorAddress) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface
func (msg MsgUpdateModeratorAddress) Type() string {
	return "update_moderator_address"
}

// GetSigners implements the sdk.Msg interface
func (msg MsgUpdateModeratorAddress) GetSigners() []sdk.AccAddress {
	moderatorAddress, err := sdk.AccAddressFromBech32(msg.ModeratorAddress)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{moderatorAddress}
}

// GetSignBytes implements the sdk.Msg interface
func (msg MsgUpdateModeratorAddress) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface
func (msg MsgUpdateModeratorAddress) ValidateBasic() error {
	if msg.ModeratorAddress == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "moderator address cannot be empty")
	}
	if msg.NewModeratorAddress == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "new moderator address cannot be empty")
	}
	if msg.ModeratorAddress == msg.NewModeratorAddress {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "moderator address and new moderator address cannot be the same")
	}
	if _, err := sdk.AccAddressFromBech32(msg.ModeratorAddress); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid moderator address (%s)", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.NewModeratorAddress); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid new moderator address (%s)", err)
	}
	return nil
}
