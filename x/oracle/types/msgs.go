package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// NewMsgRegisterOracleRequestDoc creates a new MsgRegisterOracleRequestDoc instance
func NewMsgRegisterOracleRequestDoc(
	fromAddress string,
	requestDoc OracleRequestDoc,
) *MsgRegisterOracleRequestDoc {
	return &MsgRegisterOracleRequestDoc{
		FromAddress: fromAddress,
		RequestDoc:  requestDoc,
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
	fromAddress, err := sdk.AccAddressFromBech32(msg.FromAddress)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{fromAddress}
}

// GetSignBytes implements the sdk.Msg interface
func (msg MsgRegisterOracleRequestDoc) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface
func (msg MsgRegisterOracleRequestDoc) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.FromAddress); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid from address(Moderator) (%s)", err)
	}
	if err := msg.RequestDoc.Validate(); err != nil {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	}
	return nil
}

// NewMsgUpdateOracleRequestDoc creates a new MsgUpdateOracleRequestDoc instance
func NewMsgUpdateOracleRequestDoc(
	fromAddress string,
	requestDoc OracleRequestDoc,
	reason string,
) *MsgUpdateOracleRequestDoc {
	return &MsgUpdateOracleRequestDoc{
		FromAddress: fromAddress,
		RequestDoc:  requestDoc,
		Reason:      reason,
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
	fromAddress, err := sdk.AccAddressFromBech32(msg.FromAddress)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{fromAddress}
}

// GetSignBytes implements the sdk.Msg interface
func (msg MsgUpdateOracleRequestDoc) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface
func (msg MsgUpdateOracleRequestDoc) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.FromAddress); err != nil {
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
	fromAddress string,
) *MsgSubmitOracleData {
	return &MsgSubmitOracleData{
		FromAddress: fromAddress,
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
	provider, err := sdk.AccAddressFromBech32(msg.DataSet.Provider)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{provider}
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
func NewMsgUpdateModeratorAddress(fromAddress string, moderatorAddress string) *MsgUpdateModeratorAddress {
	return &MsgUpdateModeratorAddress{
		FromAddress:      fromAddress,
		ModeratorAddress: moderatorAddress,
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
	FromAddress, err := sdk.AccAddressFromBech32(msg.FromAddress)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{FromAddress}
}

// GetSignBytes implements the sdk.Msg interface
func (msg MsgUpdateModeratorAddress) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface
func (msg MsgUpdateModeratorAddress) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.ModeratorAddress); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid moderator address (%s)", err)
	}
	return nil
}
