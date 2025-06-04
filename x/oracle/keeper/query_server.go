package keeper

import (
	"context"
	"strconv"

	"github.com/GPTx-global/guru/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ types.QueryServer = Keeper{}

// Parameters queries the parameters of the module
func (k Keeper) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	params := k.GetParams(sdk.UnwrapSDKContext(ctx))
	return &types.QueryParamsResponse{
		Params: params,
	}, nil
}

// OracleData queries oracle data by ID
func (k Keeper) OracleData(ctx context.Context, req *types.QueryOracleDataRequest) (*types.QueryOracleDataResponse, error) {
	return &types.QueryOracleDataResponse{}, nil
}

// OracleRequestDoc queries oracle request doc by ID
func (k Keeper) OracleRequestDoc(ctx context.Context, req *types.QueryOracleRequestDocRequest) (*types.QueryOracleRequestDocResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	requestId, err := strconv.ParseUint(req.RequestId, 10, 64)
	if err != nil {
		return nil, err
	}
	doc, err := k.GetOracleRequestDoc(sdkCtx, requestId)
	if err != nil {
		return nil, err
	}
	return &types.QueryOracleRequestDocResponse{
		RequestDoc: *doc,
	}, nil
}

// OracleRequestDocs queries an oracle request document list
func (k Keeper) OracleRequestDocs(ctx context.Context, req *types.QueryOracleRequestDocsRequest) (*types.QueryOracleRequestDocsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	docs := k.GetOracleRequestDocs(sdkCtx)
	return &types.QueryOracleRequestDocsResponse{
		OracleRequestDocs: docs,
	}, nil
}

// GetModeratorAddress queries the moderator address
func (k Keeper) ModeratorAddress(ctx context.Context, req *types.QueryModeratorAddressRequest) (*types.QueryModeratorAddressResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	address := k.GetModeratorAddress(sdkCtx)
	return &types.QueryModeratorAddressResponse{
		ModeratorAddress: address,
	}, nil
}
