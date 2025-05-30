package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// NewMsgRegisterOracleRequestDoc creates a new MsgRegisterOracleRequestDoc instance
func NewMsgRegisterOracleRequestDoc(
	requestDoc RequestOracleDoc,
	fee sdk.Coin,
	creator string,
	signature string,
) *MsgRegisterOracleRequestDoc {

	requestDoc.Creator = creator
	return &MsgRegisterOracleRequestDoc{
		RequestDoc: requestDoc,
		Fee:        fee,
		Creator:    creator,
		Signature:  signature,
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
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

// GetSignBytes implements the sdk.Msg interface
func (msg MsgRegisterOracleRequestDoc) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface
func (msg MsgRegisterOracleRequestDoc) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	if err := msg.RequestDoc.Validate(); err != nil {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	}
	if msg.Fee.IsZero() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "fee cannot be zero")
	}
	// if msg.Signature == "" {
	// 	return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "signature cannot be empty")
	// }
	return nil
}

// NewMsgUpdateOracleRequestDoc creates a new MsgUpdateOracleRequestDoc instance
func NewMsgUpdateOracleRequestDoc(
	requestId uint64,
	requestDoc RequestOracleDoc,
	updater string,
	signature string,
	reason string,
) *MsgUpdateOracleRequestDoc {
	return &MsgUpdateOracleRequestDoc{
		RequestId:  requestId,
		RequestDoc: requestDoc,
		Updater:    updater,
		Signature:  signature,
		Reason:     reason,
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
	updater, err := sdk.AccAddressFromBech32(msg.Updater)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{updater}
}

// GetSignBytes implements the sdk.Msg interface
func (msg MsgUpdateOracleRequestDoc) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface
func (msg MsgUpdateOracleRequestDoc) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Updater); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid updater address (%s)", err)
	}
	// if msg.RequestId == 0 {
	// 	return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "request ID cannot be empty")
	// }
	if err := msg.RequestDoc.Validate(); err != nil {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	}
	if msg.Signature == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "signature cannot be empty")
	}
	return nil
}

// NewMsgSubmitOracleData creates a new MsgSubmitOracleData instance
func NewMsgSubmitOracleData(
	requestId string,
	rawData string,
	provider string,
	signature string,
) *MsgSubmitOracleData {
	return &MsgSubmitOracleData{
		DataSet: &SubmitDataSet{
			RequestId: requestId,
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
	if msg.DataSet.RequestId == "" {
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
