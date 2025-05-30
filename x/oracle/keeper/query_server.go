package keeper

import (
	"context"

	"github.com/GPTx-global/guru/x/oracle/types"
)

var _ types.QueryServer = Keeper{}

// Parameters queries the parameters of the module
func (k Keeper) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	return &types.QueryParamsResponse{}, nil
}

// OracleData queries oracle data by ID
func (k Keeper) OracleData(ctx context.Context, req *types.QueryOracleDataRequest) (*types.QueryOracleDataResponse, error) {
	return &types.QueryOracleDataResponse{}, nil
}

// OracleRequestDoc queries oracle request doc by ID
func (k Keeper) OracleRequestDoc(ctx context.Context, req *types.QueryOracleRequestDocRequest) (*types.QueryOracleRequestDocResponse, error) {
	return &types.QueryOracleRequestDocResponse{}, nil
}

// OracleRequestDocs queries an oracle request document list
func (k Keeper) OracleRequestDocs(ctx context.Context, req *types.QueryOracleRequestDocsRequest) (*types.QueryOracleRequestDocsResponse, error) {
	return &types.QueryOracleRequestDocsResponse{}, nil
}
