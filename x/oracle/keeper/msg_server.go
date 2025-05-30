package keeper

import (
	"context"

	"github.com/GPTx-global/guru/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MsgServer implementation
var _ types.MsgServer = &Keeper{}

// RegisterOracleRequestDoc defines a method for registering a new oracle request document
func (k Keeper) RegisterOracleRequestDoc(c context.Context, doc *types.MsgRegisterOracleRequestDoc) (*types.MsgRegisterOracleRequestDocResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	// Get the current count of oracle request documents
	count := k.GetOracleRequestDocCount(ctx)

	// Create a new oracle request document
	oracleRequestDoc := types.RequestOracleDoc{
		RequestId:     count + 1,
		Status:        doc.RequestDoc.Status,
		OracleType:    doc.RequestDoc.OracleType,
		Name:          doc.RequestDoc.Name,
		Description:   doc.RequestDoc.Description,
		Period:        doc.RequestDoc.Period,
		NodeList:      doc.RequestDoc.NodeList,
		Urls:          doc.RequestDoc.Urls,
		ParseRule:     doc.RequestDoc.ParseRule,
		AggregateRule: doc.RequestDoc.AggregateRule,
		Creator:       doc.RequestDoc.Creator,
	}

	// Store the oracle request document
	k.SetOracleRequestDoc(ctx, oracleRequestDoc)

	// Increment the count
	k.SetOracleRequestDocCount(ctx, count+1)

	return &types.MsgRegisterOracleRequestDocResponse{
		RequestId: oracleRequestDoc.RequestId,
	}, nil
}

// UpdateOracleRequestDoc defines a method for updating an existing oracle request document
func (k Keeper) UpdateOracleRequestDoc(context.Context, *types.MsgUpdateOracleRequestDoc) (*types.MsgUpdateOracleRequestDocResponse, error) {
	return &types.MsgUpdateOracleRequestDocResponse{
		RequestId: 0,
	}, nil
}

// SubmitOracleData defines a method for submitting oracle data
func (k Keeper) SubmitOracleData(context.Context, *types.MsgSubmitOracleData) (*types.MsgSubmitOracleDataResponse, error) {
	return &types.MsgSubmitOracleDataResponse{}, nil

}
